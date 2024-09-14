package main

import (
	"fmt"
	"log"

	socketio "github.com/googollee/go-socket.io"
)

// ChatMessage 구조체 정의
type ChatMessage struct {
	SenderID   string `json:"senderID"`
	ReceiverID string `json:"receiverID"`
	ChatRoomID string `json:"chatRoomID"`
	Message    string `json:"message"`
	CreatedAt  string `json:"createdAt"`
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

	app.ws.OnEvent("/", "message", func(s socketio.Conn, msg ChatMessage) string {
		log.Printf("Received chat message: %v", msg)

		// 채팅방의 상대방에게 메시지 전달 (예: chatRoomID로 상대방을 찾는 로직 필요)
		if receiverConn, ok := app.users.Load(msg.ReceiverID); ok {
			log.Printf("Send Message %s to %s", msg.Message, msg.ReceiverID)
			receiverConn.(socketio.Conn).Emit("new_message", msg.Message) // 상대방에게 새 메시지를 전달
		}

		s.Emit("reply", "Message received and sent to user")
		return "Message sent to user"
	})

	app.ws.OnDisconnect("/", func(s socketio.Conn, reason string) {
		fmt.Printf("Client %s disconnected from chat: %s\n", s.ID(), reason)
		// 유저 소켓 연결을 Config의 sync.Map에서 제거
		app.users.Delete(s.ID())
	})

	go func() {
		if err := app.ws.Serve(); err != nil {
			log.Fatalf("Socket.IO server error: %v", err)
		}
	}()
}
