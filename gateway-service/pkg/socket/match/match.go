package match

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/baaami/dorandoran/broker/pkg/types"
	common "github.com/baaami/dorandoran/common/chat"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocket 연결 처리
func (app *Config) HandleMatchSocket(w http.ResponseWriter, r *http.Request) {
	// 컨텍스트 생성 및 취소 함수 정의
	ctx, cancel := context.WithCancel(context.Background())
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
		app.listenMatchEvent(ctx, conn)
	}()

	// // Ping 메시지 전송 고루틴
	// go func() {
	// 	defer wg.Done()
	// 	app.pingPump(ctx, conn)
	// }()

	// 모든 고루틴이 종료될 때까지 대기
	wg.Wait()
}

func (app *Config) listenMatchEvent(ctx context.Context, conn *websocket.Conn) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, _, err := conn.ReadMessage()
			if err != nil {
				// 클라이언트가 정상적으로 연결을 끊었을 경우 처리
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
					log.Printf("Unexpected WebSocket close error: %v", err)
				} else {
					log.Println("WebSocket connection closed by client")
				}
				return
			}
		}
	}
}

// MonitorQueue: Redis 대기열을 계속 확인하고 매칭 시도
func (app *Config) MonitorQueue(coupleCnt int) {
	matchTotalNum := coupleCnt * 2 // 남녀 비율을 고려한 총 매칭 인원수

	queueName := fmt.Sprintf("matching_queue_%d", coupleCnt) // coupleCnt에 따른 대기열 이름
	for {
		matchList, err := app.RedisClient.PopNUsersFromQueue(coupleCnt, matchTotalNum)
		if err != nil {
			log.Printf("Error in matching from queue %s: %v", queueName, err)
			continue
		}

		if len(matchList) == matchTotalNum {
			roomID := uuid.New().String()
			log.Printf("Matched %v in room %s from queue %s", matchList, roomID, queueName)

			err = app.createRoom(roomID, matchList)
			if err != nil {
				log.Printf("Failed to create room, room id: %s, err: %v", roomID, err.Error())
			}

			app.notifyUsers(matchList, roomID)
		}

		time.Sleep(2 * time.Second)
	}
}

func (app *Config) notifyUsers(matchList []string, roomID string) {
	matchMsg := common.MatchResponse{
		RoomID: roomID,
	}

	payload, err := json.Marshal(matchMsg)
	if err != nil {
		log.Printf("Failed to marshal match response: %v", err)
		return
	}

	webSocketMsg := WebSocketMessage{
		Type:    MessageTypeMatch,
		Status:  PushMessageStatusMatchSuccess,
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

// [Hub Network] Chat 서비스에 API를 호출하여 방 생성
func (app *Config) createRoom(roomID string, matchList []string) error {
	client := &http.Client{
		Timeout: time.Second * 10, // 요청 타임아웃 설정
	}

	chatRoom := ChatRoom{
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

	// matchFilter에 따른 처리가 필요함
	app.RedisClient.AddUserToQueue(userID, matchFilter.CoupleCount)
	log.Printf("User %s added to waiting queue", userID)

	return nil
}

// UnRegister 메시지 처리
func (app *Config) UnRegisterMatchClient(userID string) {
	app.MatchClients.Delete(userID)

	_, err := app.RedisClient.PopUserFromQueue(userID)
	if err != nil {
		log.Printf("fail pop %s user from redis queue", userID)
	}
	log.Printf("User %s unregister match server", userID)
}

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
