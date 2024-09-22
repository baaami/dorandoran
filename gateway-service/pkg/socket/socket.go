package socket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/baaami/dorandoran/broker/event"
	"github.com/baaami/dorandoran/broker/pkg/redis"
	"github.com/gorilla/websocket"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	MessageTypeChat     = "chat"
	MessageTypeMatch    = "match"
	MessageTypeRegister = "register"
)

type Config struct {
	Clients     sync.Map
	Rabbit      *amqp.Connection
	RedisClient *redis.RedisClient
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type RegisterMessage struct {
	UserID string `json:"user_id"`
}

type MatchMessage struct {
	UserID string `json:"user_id"`
}

type ChatMessage struct {
	RoomID     string `json:"room_id"`
	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
	Message    string `json:"message"`
}

// WebSocket 연결 처리
func (app *Config) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	var regiMsg RegisterMessage
	var userID string

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			if userID != "" {
				app.Clients.Delete(userID)
			}
			return
		}

		var wsMsg WebSocketMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		switch wsMsg.Type {
		case MessageTypeRegister:
			if err := json.Unmarshal(wsMsg.Payload, &regiMsg); err != nil {
				log.Printf("Failed to unmarshal register message: %v", err)
				continue
			}

			userID = regiMsg.UserID
			app.Clients.Store(userID, conn)
			log.Printf("User %s registered", userID)

			// 유저를 대기열에 추가하고 매칭을 시도
			go app.RedisClient.AddUserToQueue(userID)

		case MessageTypeChat:
			var chatMsg ChatMessage
			if err := json.Unmarshal(wsMsg.Payload, &chatMsg); err != nil {
				log.Printf("Failed to unmarshal chat message: %v", err)
				continue
			}

			log.Printf("chatMsg %v", chatMsg)
			app.HandleChatMessage(chatMsg)
		}
	}
}

// Redis 대기열을 계속 확인하고 매칭 시도
func (app *Config) MonitorQueue() {
	for {
		user1, user2, err := app.RedisClient.PopUsersFromQueue()
		if err != nil {
			log.Printf("Error in matching: %v", err)
			continue
		}

		if user1 != "" && user2 != "" {
			roomID := user1 + "-" + user2
			log.Printf("Matched %s with %s in room %s", user1, user2, roomID)
			app.notifyUsers(user1, user2, roomID)
		}

		time.Sleep(2 * time.Second)
	}
}

func (app *Config) notifyUsers(user1, user2, roomID string) {
	matchMsg := MatchMessage{
		UserID: roomID,
	}

	for _, userID := range []string{user1, user2} {
		if conn, ok := app.Clients.Load(userID); ok {
			conn.(*websocket.Conn).WriteJSON(matchMsg)
			log.Printf("Notified %s about match in room %s", userID, roomID)
		} else {
			log.Printf("User %s not connected", userID)
		}
	}
}

// 채팅 메시지 처리
func (app *Config) HandleChatMessage(chatMsg ChatMessage) {
	log.Printf("Received chat message from %s: %s", chatMsg.SenderID, chatMsg.Message)

	if receiverConn, ok := app.Clients.Load(chatMsg.ReceiverID); ok {
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
