package socket

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	common "github.com/baaami/dorandoran/common/chat"
	"github.com/gorilla/websocket"
)

type JoinRoomMessage struct {
	RoomID string `json:"room_id"`
}

type LeaveRoomMessage struct {
	RoomID string `json:"room_id"`
}

// WebSocket 연결 처리
func (app *Config) HandleChatSocket(w http.ResponseWriter, r *http.Request) {
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// TODO: 다중 인스턴스 환경에서의 세션 관리나 메시지 전달을 위해 Redis 같은 중앙 집중식 저장소를 활용하는 것을 고려
	// TODO: 클라이언트와의 연결을 주기적으로 확인하여, 비정상적으로 종료된 연결을 감지하고 정리하는 메커니즘(예: 핑-퐁 메시지)을 도입
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
	app.RegisterChatClient(conn, userID)
	defer func() {
		app.UnRegisterChatClient(userID)
		conn.Close()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			// 클라이언트가 정상적으로 연결을 끊었을 경우 처리
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				log.Printf("Unexpected WebSocket close error: %v", err)
			} else {
				log.Println("WebSocket connection closed by client")
			}
			break
		}

		var wsMsg common.WebSocketMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		switch wsMsg.Type {
		case MessageTypeBroadCast:
			app.handleBroadCastMessage(wsMsg.Payload, userID)
		case MessageTypeJoin:
			app.handleJoinMessage(wsMsg.Payload, userID)
		case MessageTypeLeave:
			app.handleLeaveMessage(wsMsg.Payload, userID)
		}
	}
}

// BroadCast 메시지 처리
func (app *Config) handleBroadCastMessage(payload json.RawMessage, userID string) {
	var broadCastMsg ChatMessage
	if err := json.Unmarshal(payload, &broadCastMsg); err != nil {
		log.Printf("Failed to unmarshal broadcast message: %v", err)
		return
	}

	chat := Chat{
		RoomID:    broadCastMsg.RoomID,
		SenderID:  userID,
		Message:   broadCastMsg.Message,
		CreatedAt: time.Now(),
	}

	app.BroadcastToRoom(chat)
}

// Join 메시지 처리
func (app *Config) handleJoinMessage(payload json.RawMessage, userID string) {
	var joinMsg JoinRoomMessage
	if err := json.Unmarshal(payload, &joinMsg); err != nil {
		log.Printf("Failed to unmarshal join message: %v, err: %v", payload, err)
		return
	}

	app.JoinRoom(joinMsg.RoomID, userID)
}

// Leave 메시지 처리
func (app *Config) handleLeaveMessage(payload json.RawMessage, userID string) {
	var leaveMsg LeaveRoomMessage
	if err := json.Unmarshal(payload, &leaveMsg); err != nil {
		log.Printf("Failed to unmarshal leave message: %v, err: %v", payload, err)
		return
	}

	app.LeaveRoom(leaveMsg.RoomID, userID)
}

// Register
func (app *Config) RegisterChatClient(conn *websocket.Conn, userID string) {
	client := &Client{
		Conn: conn,
		Send: make(chan interface{}, 256),
	}

	// 쓰기 고루틴 시작
	go client.writePump()

	app.ChatClients.Store(userID, client)
	log.Printf("User %s register chat server", userID)
}

// UnRegister
func (app *Config) UnRegisterChatClient(userID string) {
	if clientInterface, ok := app.ChatClients.Load(userID); ok {
		client := clientInterface.(*Client)

		// Send 채널 닫기
		close(client.Send)

		// Channel에서 유저 제거
		app.ChatClients.Delete(userID)

		log.Printf("User %s unregistered chat server", userID)
	}
}

func (c *Client) writePump() {
	defer func() {
		c.Conn.Close()
		log.Printf("[INFO] writePump for user %v exited", c.Conn.RemoteAddr())
	}()

	for {
		message, ok := <-c.Send
		if !ok {
			// 채널이 닫힌 경우 연결을 닫습니다.
			c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		// 메시지를 전송합니다.
		if err := c.Conn.WriteJSON(message); err != nil {
			log.Printf("[ERROR] Failed to write message: %v", err)
			return
		}
	}
}
