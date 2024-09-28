package socket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type MatchResponse struct {
	RoomID string `json:"room_id"`
}

// WebSocket 연결 처리
func (app *Config) HandleMatchSocket(w http.ResponseWriter, r *http.Request) {
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
	defer conn.Close()

	// URL에서 유저 ID 가져오기
	userID := r.Header.Get("X-User-ID")
	app.RegisterMatchClient(conn, userID)

	for {
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

// Redis 대기열을 계속 확인하고 매칭 시도
func (app *Config) MonitorQueue() {
	const MatchTotalNum = 2

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
	matchMsg := MatchResponse{
		RoomID: roomID,
	}

	log.Printf("Match Notify Start!!!")

	for _, userID := range matchList {
		log.Printf("Try to notify user, %s", userID)

		if conn, ok := app.MatchClients.Load(userID); ok {
			conn.(*websocket.Conn).WriteJSON(matchMsg)
			log.Printf("Notified %s about match in room %s", userID, roomID)

			conn.(*websocket.Conn).Close()
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
	app.MatchClients.Store(userID, conn)
	log.Printf("User %s register match server", userID)

	app.RedisClient.AddUserToQueue(userID)
	log.Printf("User %s added to waiting queue", userID)
}

// UnRegister 메시지 처리
func (app *Config) UnRegisterMatchClient(userID string) {
	app.MatchClients.Delete(userID)
	log.Printf("User %s unregister match server", userID)
}
