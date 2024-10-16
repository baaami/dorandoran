package socket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

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

	app.RegisterMatchClient(conn, userID)

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

// Redis 대기열을 계속 확인하고 매칭 시도
func (app *Config) MonitorQueue() {
	const MatchTotalNum = 2

	// TODO: 존재하는 모든 대기열에 대해서 모니터링 해야함
	// -> 현재는 1:1 매칭으로 1개의 큐로 관리
	for {
		matchList, err := app.RedisClient.PopNUsersFromQueue(MatchTotalNum)
		if err != nil {
			log.Printf("Error in matching: %v", err)
			continue
		}

		if len(matchList) == MatchTotalNum {
			roomID := uuid.New().String()
			log.Printf("Matched %v in room %s", matchList, roomID)

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
func (app *Config) RegisterMatchClient(conn *websocket.Conn, userID string) {
	// 존재하는 사람의 경우 pass
	app.MatchClients.Store(userID, conn)
	log.Printf("User %s register match server", userID)

	app.RedisClient.AddUserToQueue(userID, 1)
	log.Printf("User %s added to waiting queue", userID)
}

// UnRegister 메시지 처리
func (app *Config) UnRegisterMatchClient(userID string) {
	app.MatchClients.Delete(userID)
	log.Printf("User %s unregister match server", userID)

	// TODO: RedisClient에 존재할 경우 삭제해줘야한다.
}
