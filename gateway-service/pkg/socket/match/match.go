package match

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/baaami/dorandoran/broker/pkg/data"
	"github.com/baaami/dorandoran/broker/pkg/types"
	common "github.com/baaami/dorandoran/common/user"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocket 연결 처리
func (app *Config) HandleMatchSocket(w http.ResponseWriter, r *http.Request) {
	// 30초 타임아웃 설정
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// 클라이언트가 정상적으로 연결을 끊었을 경우 처리
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			log.Printf("Unexpected WebSocket close error: %v", err)
		} else {
			log.Println("WebSocket connection closed by client")
		}
		return
	}

	// URL에서 유저 ID 가져오기
	userID := r.Header.Get("X-User-ID")

	// MatchClients에서 이미 존재하는지 확인
	if _, exists := app.MatchClients.Load(userID); exists {
		log.Printf("User %s is already Register Matching Queue", userID)

		// 이미 등록된 경우 409 Conflict 상태 코드 반환
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(fmt.Sprintf("User %s is already in the matching queue", userID)))
		return
	}

	defer func() {
		app.UnRegisterMatchClient(userID)
		conn.Close()
	}()

	err = app.RegisterMatchClient(conn, userID)
	if err != nil {
		log.Printf("Failed to register match queue, err: %s", err.Error())
	}

	// WaitGroup을 사용하여 모든 고루틴이 종료될 때까지 대기
	var wg sync.WaitGroup
	wg.Add(1) // 두 개의 고루틴 (listenChatEvent, pingPump)

	// 메시지 처리 고루틴
	go func() {
		defer wg.Done()
		app.listenMatchEvent(ctx, conn, userID)
	}()

	// // Ping 메시지 전송 고루틴
	// go func() {
	// 	defer wg.Done()
	// 	app.pingPump(ctx, conn)
	// }()

	// 모든 고루틴이 종료될 때까지 대기
	wg.Wait()
}

// 매칭 서버 메시지 처리 함수
func (app *Config) listenMatchEvent(ctx context.Context, conn *websocket.Conn, userID string) {
	for {
		// 컨텍스트의 타임아웃을 웹소켓 연결에 적용
		deadline, ok := ctx.Deadline()
		if ok {
			conn.SetReadDeadline(deadline)
		}

		select {
		case <-ctx.Done():
			// 30초 타임아웃 시 매칭 실패 메시지 전송
			if ctx.Err() == context.DeadlineExceeded {
				log.Printf("Matching timed out for user %s", userID)
				app.sendMatchFailureMessage(conn)
			}
			return
		default:
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
					log.Printf("Unexpected WebSocket close error: %v", err)
				} else if ctx.Err() == context.DeadlineExceeded || isTimeoutError(err) {
					log.Printf("WebSocket read timeout for user %s", userID)
					app.sendMatchFailureMessage(conn)
					continue
				} else {
					log.Println("WebSocket connection closed by client")
				}
				return
			}
		}
	}
}

// 타임아웃 에러 확인 함수
func isTimeoutError(err error) bool {
	netErr, ok := err.(net.Error)
	return ok && netErr.Timeout()
}

// Register 메시지 처리
func (app *Config) RegisterMatchClient(conn *websocket.Conn, userID string) error {
	_, ok := app.MatchClients.Load(userID)
	if ok {
		log.Printf("User %s already registered match server", userID)
		return fmt.Errorf("user %s already registered match server", userID)
	}

	// TODO: 존재하는 사람의 경우 pass, 중복 처리 필요
	app.MatchClients.Store(userID, conn)
	log.Printf("User %s register match server", userID)

	matchFilter, err := GetMatchFilter(userID)
	if err != nil {
		log.Printf("Failed to get matchfilter, user: %s", userID)
		return err
	}

	user, err := GetUserInfo(userID)
	if err != nil {
		log.Printf("Failed to get GetUserInfo, user: %s", userID)
		return err
	}

	waitingUser := types.WaitingUser{
		ID:              user.ID,
		Gender:          user.Gender,
		Birth:           user.Birth,
		Address:         types.Address(user.Address),
		AddressRangeUse: matchFilter.AddressRangeUse,
		AgeGroupUse:     matchFilter.AgeGroupUse,
	}

	app.RedisClient.AddUserToQueue(matchFilter.CoupleCount, waitingUser)

	log.Printf("User %s added to waiting queue", userID)

	return nil
}

// UnRegister 메시지 처리
func (app *Config) UnRegisterMatchClient(userID string) {
	app.MatchClients.Delete(userID)

	matchFilter, err := GetMatchFilter(userID)
	if err != nil {
		log.Printf("Failed to get matchfilter, user: %s", userID)
		return
	}

	user, err := GetUserInfo(userID)
	if err != nil {
		log.Printf("Failed to get GetUserInfo, user: %s", userID)
		return
	}

	// 매칭되어 종료될 경우 존재하지 않을 수도 있음
	err = app.RedisClient.PopUserFromQueue(user.ID, matchFilter.CoupleCount)
	if err != nil {
		log.Printf("fail pop %s user from redis queue", userID)
	}
	log.Printf("User %s unregister match server", userID)
}

// 대기열 모니터링
func (app *Config) MonitorQueue(coupleCnt int) {
	matchTotalNum := coupleCnt * 2 // 총 매침 인원 수

	for {
		// Redis에서 대기열 모니터링 처리
		matchIDList, err := app.RedisClient.MonitorAndPopMatchingUsers(coupleCnt)
		if err != nil || len(matchIDList) < matchTotalNum {
			time.Sleep(2 * time.Second)
			continue
		}

		// 매침 성공 시 사용자 알림 및 방 생성
		if len(matchIDList) == matchTotalNum {
			roomID := uuid.New().String()
			log.Printf("Matched room %s", roomID)

			err = app.createRoom(roomID, matchIDList)
			if err != nil {
				log.Printf("Failed to create room, room id: %s, err: %v", roomID, err.Error())
			}

			app.sendMatchSuccessMessage(matchIDList, roomID)
		}

		// 일정 주기만큼 실행
		time.Sleep(2 * time.Second)
	}
}

// 매칭 성공 메시지 전송 함수
func (app *Config) sendMatchSuccessMessage(matchList []string, roomID string) {
	matchMsg := MatchResponse{
		Type:   PushMessageStatusMatchSuccess,
		RoomID: roomID,
	}

	payload, err := json.Marshal(matchMsg)
	if err != nil {
		log.Printf("Failed to marshal match response: %v", err)
		return
	}

	webSocketMsg := WebSocketMessage{
		Kind:    MessageTypeMatch,
		Payload: json.RawMessage(payload),
	}

	log.Printf("Match Notify Start!!!")

	for _, userID := range matchList {
		log.Printf("Try to notify user, %s", userID)

		if conn, ok := app.MatchClients.Load(userID); ok {
			err := conn.(*websocket.Conn).WriteJSON(webSocketMsg)
			if err != nil {
				log.Printf("Failed to notify user %s: %v", userID, err)
			} else {
				log.Printf("Notified %s about match in room %s", userID, roomID)
			}
		} else {
			log.Printf("User %s not connected", userID)
		}
	}

	log.Printf("Match Notify End!!!")
}

// 매칭 실패 메시지 전송 함수
func (app *Config) sendMatchFailureMessage(conn *websocket.Conn) {
	matchMsg := MatchResponse{
		Type:   PushMessageStatusMatchFailure,
		RoomID: "",
	}

	payload, err := json.Marshal(matchMsg)
	if err != nil {
		log.Printf("Failed to marshal match response: %v", err)
		return
	}

	webSocketMsg := WebSocketMessage{
		Kind:    MessageTypeMatch,
		Payload: json.RawMessage(payload),
	}

	if err := conn.WriteJSON(webSocketMsg); err != nil {
		log.Printf("Failed to send match failure message: %v", err)
	}
}

// [Bridge chat] 방 생성
func (app *Config) createRoom(roomID string, matchList []string) error {
	client := &http.Client{
		Timeout: time.Second * 10, // 요청 타임아웃 설정
	}

	chatRoom := data.ChatRoom{
		ID:    roomID,
		Users: matchList,
	}

	reqBody, err := json.Marshal(chatRoom)
	if err != nil {
		log.Printf("Failed to marshal chatroom, chatroom: %v, err: %s", chatRoom, err.Error())
		return nil
	}

	// 요청 URL 생성
	url := "http://chat-service/room/create"

	// GET 요청 생성
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// 요청 실행
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("failed to send request: %v", err)
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// 응답 처리
	if resp.StatusCode != http.StatusCreated {
		// room이 생성되지 않은 경우
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// [Bridge user] 유저 매칭 필터 조회
func GetMatchFilter(userID string) (*types.MatchFilter, error) {
	var matchFilter types.MatchFilter

	// Matching 필터 획득
	client := http.Client{}

	req, err := http.NewRequest(http.MethodGet, "http://user-service/match/filter", nil)
	if err != nil {
		return nil, err
	}

	// 사용자 ID를 요청의 헤더에 추가
	req.Header.Set("X-User-ID", userID)

	// 요청 실행
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	err = json.Unmarshal(body, &matchFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &matchFilter, nil
}

// [Bridge user] 유저 정보 조회
func GetUserInfo(userID string) (*common.User, error) {
	var user common.User

	// Matching 필터 획득
	client := http.Client{}

	req, err := http.NewRequest(http.MethodGet, "http://user-service/find", nil)
	if err != nil {
		return nil, err
	}

	// 사용자 ID를 요청의 헤더에 추가
	req.Header.Set("X-User-ID", userID)

	// 요청 실행
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	err = json.Unmarshal(body, &user)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &user, nil
}
