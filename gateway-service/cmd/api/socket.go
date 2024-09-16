package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/baaami/dorandoran/broker/event"
	socketio "github.com/googollee/go-socket.io"
)

// ChatMessage 구조체 정의
type ChatMessage struct {
	RoomID     string    `bson:"room_id"`
	SenderID   string    `bson:"sender_id"`
	ReceiverID string    `bson:"receiver_id"`
	Message    string    `bson:"message"`
	CreatedAt  time.Time `bson:"created_at"`
}

func (app *Config) RegisterSocketServer() {
	app.ws = socketio.NewServer(nil)

	app.ws.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		fmt.Println("connected to chat id:", s.ID())
		// 유저 ID와 소켓 연결을 Config의 sync.Map에 저장
		app.users.Store(s.ID(), s)
		return nil
	})

	app.ws.OnEvent("/", "message", func(s socketio.Conn, chatMsg ChatMessage) string {
		log.Printf("Received chat message: %v", chatMsg)

		// 채팅방의 상대방에게 메시지 전달 (예: chatRoomID로 상대방을 찾는 로직 필요)
		if receiverConn, ok := app.users.Load(chatMsg.ReceiverID); ok {
			log.Printf("Send Message %s to %s", chatMsg.Message, chatMsg.ReceiverID)
			receiverConn.(socketio.Conn).Emit("new_message", chatMsg.Message) // 상대방에게 새 메시지를 전달

			// push rabbitmq
			app.pushChatToQueue(chatMsg)
		}

		s.Emit("reply", "Message received and sent to user")
		return "Message sent to user"
	})

	app.ws.OnDisconnect("/", func(s socketio.Conn, reason string) {
		fmt.Printf("Client %s disconnected from chat: %s\n", s.ID(), reason)
		// 유저 소켓 연결을 Config의 sync.Map에서 제거
		app.users.Delete(s.ID())
	})

	app.ws.OnError("/", func(s socketio.Conn, e error) {
		log.Printf("Error on client %s: %v", s.ID(), e)
	})

	go func() {
		if err := app.ws.Serve(); err != nil {
			log.Fatalf("Socket.IO server error: %v", err)
		}
	}()
}

// pushChatToQueue pushes a message into RabbitMQ
func (app *Config) pushChatToQueue(chatMsg ChatMessage) error {
	if app.Rabbit == nil {
		log.Println("RabbitMQ connection is nil")
		return fmt.Errorf("RabbitMQ connection is nil")
	}

	emitter, err := event.NewEventEmitter(app.Rabbit)
	if err != nil {
		return err
	}

	payload := ChatMessage{
		SenderID:   chatMsg.SenderID,
		ReceiverID: chatMsg.ReceiverID,
		RoomID:     chatMsg.RoomID,
		Message:    chatMsg.Message,
		CreatedAt:  chatMsg.CreatedAt,
	}

	j, _ := json.MarshalIndent(&payload, "", "\t")
	err = emitter.Push(string(j), "chat")
	if err != nil {
		log.Printf("Failed to push message to queue: %v", err)
		return err
	}
	return nil
}
