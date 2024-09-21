// socket.go
package socket

import (
	"encoding/json"
	"log"
	"sync"

	"net/http"

	"github.com/baaami/dorandoran/broker/event"
	"github.com/gorilla/websocket"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	MessageTypeChat     = "chat"
	MessageTypeMatch    = "match"
	MessageTypeRegister = "register"
)

type Config struct {
	Clients sync.Map
	Rabbit  *amqp.Connection
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
