package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/baaami/dorandoran/broker/event"
)

// 메시지 타입 정의
const (
	MessageTypeChat     = "chat"
	MessageTypeMatch    = "match"
	MessageTypeRegister = "register" // 유저 등록 메시지
)

type RegisterMessage struct {
	UserID string `json:"user_id"`
}

// WebSocket 메시지 구조체 정의
type WebSocketMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// 채팅 메시지 구조체 정의
type ChatMessage struct {
	RoomID     string `json:"room_id"`
	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
	Message    string `json:"message"`
}

// WebSocket 연결 처리
func (app *Config) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	var regiMsg RegisterMessage
	var userID string // 접속한 유저의 ID

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			// 유저가 연결 해제되면 해당 유저를 메모리에서 제거
			if userID != "" {
				app.mu.Lock()
				delete(app.clients, userID)
				app.mu.Unlock()
			}
			return
		}

		// WebSocket 메시지 처리
		var wsMsg WebSocketMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		switch wsMsg.Type {
		case MessageTypeRegister:
			// 유저 등록 처리
			if err := json.Unmarshal(wsMsg.Payload, &regiMsg); err != nil {
				log.Printf("Failed to unmarshal register message: %v", err)
				continue
			}

			userID = regiMsg.UserID

			app.mu.Lock()
			app.clients[userID] = conn
			app.mu.Unlock()
			log.Printf("User %s registered", userID)

		case MessageTypeChat:
			// 채팅 메시지 처리
			var chatMsg ChatMessage
			if err := json.Unmarshal(wsMsg.Payload, &chatMsg); err != nil {
				log.Printf("Failed to unmarshal chat message: %v", err)
				continue
			}

			log.Printf("chatMsg %v", chatMsg)
			app.handleChatMessage(chatMsg)
		}
	}
}

// 채팅 메시지 처리
func (app *Config) handleChatMessage(chatMsg ChatMessage) {
	log.Printf("Received chat message from %s: %s", chatMsg.SenderID, chatMsg.Message)

	// 수신자에게 메시지 전달
	app.mu.Lock()
	if receiverConn, ok := app.clients[chatMsg.ReceiverID]; ok {
		log.Printf("Sending message to %s", chatMsg.ReceiverID)
		receiverConn.WriteJSON(chatMsg)

		// 메시지를 RabbitMQ에 푸시
		emitter, err := event.NewEventEmitter(app.Rabbit)
		if err == nil {
			emitter.PushChatMessageToQueue(event.ChatMessage(chatMsg))
		}
	} else {
		log.Printf("Receiver %s not connected", chatMsg.ReceiverID)
	}
	app.mu.Unlock()
}
