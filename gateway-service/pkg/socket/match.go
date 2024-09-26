package socket

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	common "github.com/baaami/dorandoran/common/chat"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type MatchMessage struct {
	UserID string `json:"user_id"`
}

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
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		log.Printf("Failed to Atoi user ID, err: %s", err.Error())
		http.Error(w, "Failed to Atoi user ID", http.StatusInternalServerError)
		return
	}

	app.RegisterMatchClient(conn, userID)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			// 클라이언트가 정상적으로 연결을 끊었을 경우 처리
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				log.Printf("Unexpected WebSocket close error: %v", err)
			} else {
				log.Println("WebSocket connection closed by client")
			}
			return
		}

		var wsMsg common.WebSocketMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		switch wsMsg.Type {
		case MessageTypeMatch:
			app.handleMatchMessage(wsMsg.Payload)
		}
	}
}

// Match 메시지 처리
func (app *Config) handleMatchMessage(payload json.RawMessage) {
	var matchMsg MatchMessage
	if err := json.Unmarshal(payload, &matchMsg); err != nil {
		log.Printf("Failed to unmarshal match message: %v", err)
		return
	}

	app.RedisClient.AddUserToQueue(matchMsg.UserID)
	log.Printf("User %s added to waiting queue", matchMsg.UserID)
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
			app.notifyUsers(matchList, roomID)
		}

		time.Sleep(2 * time.Second)
	}
}

func (app *Config) notifyUsers(matchList []string, roomID string) {
	matchMsg := MatchResponse{
		RoomID: roomID,
	}

	for _, userID := range matchList {
		if conn, ok := app.MatchClients.Load(userID); ok {
			conn.(*websocket.Conn).WriteJSON(matchMsg)
			log.Printf("Notified %s about match in room %s", userID, roomID)
		} else {
			log.Printf("User %s not connected", userID)
		}
	}
}

// Register 메시지 처리
func (app *Config) RegisterMatchClient(conn *websocket.Conn, userID int) {
	app.MatchClients.Store(userID, conn)
	log.Printf("User %d register match server", userID)
}

// UnRegister 메시지 처리
func (app *Config) UnRegisterMatchClient(userID int) {
	app.MatchClients.Delete(userID)
	log.Printf("User %d unregister match server", userID)
}
