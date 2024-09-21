// socket.go
package socket

import (
	"encoding/json"
	"log"
	"sync"

	"net/http"

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

		case MessageTypeChat:
			var chatMsg ChatMessage
			if err := json.Unmarshal(wsMsg.Payload, &chatMsg); err != nil {
				log.Printf("Failed to unmarshal chat message: %v", err)
				continue
			}

			log.Printf("chatMsg %v", chatMsg)
			app.HandleChatMessage(chatMsg)

		case MessageTypeMatch:
			var matchMsg MatchMessage
			if err := json.Unmarshal(wsMsg.Payload, &matchMsg); err != nil {
				log.Printf("Failed to unmarshal match message: %v", err)
				continue
			}
			app.HandleMatchRequest(matchMsg, conn)
		}
	}
}

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

// HandleMatchRequest 매칭 요청 처리
func (app *Config) HandleMatchRequest(matchMsg MatchMessage, conn *websocket.Conn) {
	// Redis 대기열에 유저 추가
	err := app.RedisClient.AddUserToQueue(matchMsg.UserID)
	if err != nil {
		log.Printf("Failed to add user to Redis queue: %v", err)
		return
	}
	log.Printf("User %s added to match queue", matchMsg.UserID)

	waitingClients, err := app.RedisClient.GetAllUsersInQueue()
	if err != nil {
		log.Printf("Failed to get users from Redis queue: %v", err)
		return
	}

	log.Printf("Waiting Clients %v", waitingClients)

	// 매칭을 위해 두 명의 유저를 대기열에서 가져옴
	user1, user2, err := app.RedisClient.PopUsersFromQueue()
	if err != nil {
		log.Printf("Failed to pop users from queue: %v", err)
		return
	}

	if user1 != "" && user2 != "" {
		// 매칭 성공
		roomID := user1 + "-" + user2
		matchResponse := map[string]string{
			"user1_id": user1,
			"user2_id": user2,
			"room_id":  roomID,
		}

		// 유저1에게 매칭 정보 전달
		if conn1, ok := app.Clients.Load(user1); ok {
			conn1.(*websocket.Conn).WriteJSON(matchResponse)
		}

		// 유저2에게 매칭 정보 전달
		if conn2, ok := app.Clients.Load(user2); ok {
			conn2.(*websocket.Conn).WriteJSON(matchResponse)
		}

		log.Printf("Match made between %s and %s in room %s", user1, user2, roomID)
	}
}
