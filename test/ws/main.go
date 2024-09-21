package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

type RegisterMessage struct {
	UserID string `json:"user_id"`
}

type ChatMessage struct {
	RoomID     string `json:"room_id"`
	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
	Message    string `json:"message"`
}

type WebSocketMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func main() {
	// WebSocket 서버에 연결
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:2719/ws", nil)
	if err != nil {
		log.Fatalf("Failed to connect to WebSocket server: %v", err)
	}
	defer conn.Close()

	fmt.Println("Connected to WebSocket server!")

	// 콘솔에서 사용자 입력 처리
	reader := bufio.NewReader(os.Stdin)

	// 유저 ID 입력받기
	fmt.Print("Enter your User ID: ")
	userID, _ := reader.ReadString('\n')
	userID = strings.TrimSpace(userID) // 공백 제거

	regiMsg := RegisterMessage{
		UserID: userID,
	}

	// 유저 ID를 서버에 등록하는 메시지 전송
	userIDBytes, err := json.Marshal(regiMsg) // 유저 ID를 JSON으로 변환
	if err != nil {
		log.Fatalf("Failed to marshal user ID: %v", err)
	}

	// 유저 ID를 서버에 등록하는 메시지 전송
	registerMsg := WebSocketMessage{
		Type:    "register",
		Payload: userIDBytes,
	}

	registerMsgBytes, err := json.Marshal(registerMsg)
	if err != nil {
		log.Fatalf("Failed to marshal register message: %v", err)
	}

	// 서버에 유저 등록
	err = conn.WriteMessage(websocket.TextMessage, registerMsgBytes)
	if err != nil {
		log.Fatalf("Failed to send register message: %v", err)
	}
	fmt.Printf("User %s registered with server\n", userID)

	// 채팅방 정보 입력받기
	fmt.Print("Enter Chat Room ID: ")
	roomID, _ := reader.ReadString('\n')
	roomID = strings.TrimSpace(roomID)

	fmt.Print("Enter Receiver User ID: ")
	receiverID, _ := reader.ReadString('\n')
	receiverID = strings.TrimSpace(receiverID)

	// 채팅 메시지 수신 처리 (고루틴으로 실행)
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Error reading message: %v", err)
				return
			}

			// 수신된 메시지를 출력
			var chatMsg ChatMessage
			if err := json.Unmarshal(msg, &chatMsg); err != nil {
				log.Printf("Failed to unmarshal received message: %v", err)
				continue
			}

			// 수신된 메시지 출력
			fmt.Printf("\nNew message from %s: %s\n", chatMsg.SenderID, chatMsg.Message)
		}
	}()

	fmt.Println("You can now start chatting. Type 'quit' to exit.")

	// 메시지 입력 및 전송
	for {
		fmt.Print("Enter message: ")
		msgText, _ := reader.ReadString('\n')
		msgText = strings.TrimSpace(msgText)

		if msgText == "quit" {
			fmt.Println("Disconnecting...")
			break
		}

		// 채팅 메시지 생성
		chatMsg := ChatMessage{
			SenderID:   userID,
			ReceiverID: receiverID,
			RoomID:     roomID,
			Message:    msgText,
		}

		// ChatMessage를 JSON으로 변환하여 Payload로 설정
		chatMsgBytes, err := json.Marshal(chatMsg)
		if err != nil {
			log.Printf("Failed to marshal chat message: %v", err)
			continue
		}

		// WebSocketMessage 생성
		wsMsg := WebSocketMessage{
			Type:    "chat",
			Payload: chatMsgBytes,
		}

		// 메시지를 JSON으로 인코딩
		wsMsgBytes, err := json.Marshal(wsMsg)
		if err != nil {
			log.Printf("Failed to marshal chat message: %v", err)
			continue
		}

		// WebSocket 서버로 메시지 전송
		err = conn.WriteMessage(websocket.TextMessage, wsMsgBytes)
		if err != nil {
			log.Printf("Failed to send message: %v", err)
			break
		}
	}

	fmt.Println("Client exiting.")
}
