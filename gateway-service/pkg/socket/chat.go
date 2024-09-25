package socket

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/baaami/dorandoran/broker/event"
	common "github.com/baaami/dorandoran/common/chat"
	"github.com/gorilla/websocket"
)

const (
	MessageTypeChat       = "chat"
	MessageTypeMatch      = "match"
	MessageTypeRegister   = "register"
	MessageTypeUnRegister = "unregister"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type MatchMessage struct {
	UserID string `json:"user_id"`
}

type MatchResponse struct {
	RoomID string `json:"room_id"`
}

// WebSocket 연결 처리
func (app *Config) HandleChatSocket(w http.ResponseWriter, r *http.Request) {
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

	// 연결 성공
	defer conn.Close()

	// URL에서 유저 ID 가져오기
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		log.Printf("Failed to Atoi user ID, err: %s", err.Error())
		http.Error(w, "Failed to Atoi user ID", http.StatusInternalServerError)
		return
	}

	app.RegisterChatClient(conn, userID)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			// 클라이언트가 정상적으로 연결을 끊었을 경우 처리
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				log.Printf("Unexpected WebSocket close error: %v", err)
			} else {
				log.Println("WebSocket connection closed by client")
			}

			app.UnRegisterChatClient(userID)
			return
		}

		var wsMsg common.WebSocketMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		switch wsMsg.Type {
		case MessageTypeChat:
			app.handleChatMessage(wsMsg.Payload)
		}
	}
}

// Chat 메시지 처리
func (app *Config) handleChatMessage(payload json.RawMessage) {
	var chatMsg common.ChatMessage
	if err := json.Unmarshal(payload, &chatMsg); err != nil {
		log.Printf("Failed to unmarshal chat message: %v", err)
		return
	}

	log.Printf("Received chat message from %s: %s", chatMsg.SenderID, chatMsg.Message)

	if receiverConn, ok := app.ChatClients.Load(chatMsg.ReceiverID); ok {
		conn := receiverConn.(*websocket.Conn)
		log.Printf("Sending message to %s", chatMsg.ReceiverID)
		conn.WriteJSON(chatMsg)

		emitter, err := event.NewEventEmitter(app.Rabbit)
		if err == nil {
			emitter.PushChatMessageToQueue(event.ChatMessage(chatMsg))
		}
	} else {
		log.Printf("Receiver %s not connected", chatMsg.ReceiverID)
	}
}

// Register 메시지 처리
func (app *Config) RegisterChatClient(conn *websocket.Conn, userID int) {
	app.ChatClients.Store(userID, conn)
	log.Printf("User %d register chat server", userID)
}

// UnRegister 메시지 처리
func (app *Config) UnRegisterChatClient(userID int) {
	app.ChatClients.Delete(userID)
	log.Printf("User %d unregister chat server", userID)
}
