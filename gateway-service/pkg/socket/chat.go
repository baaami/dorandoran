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

// WebSocket 연결 처리
func (app *Config) HandleChatSocket(w http.ResponseWriter, r *http.Request) {
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

	app.RegisterChatClient(conn, strconv.Itoa(userID))

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			// 클라이언트가 정상적으로 연결을 끊었을 경우 처리
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				log.Printf("Unexpected WebSocket close error: %v", err)
			} else {
				log.Println("WebSocket connection closed by client")
			}

			app.UnRegisterChatClient(strconv.Itoa(userID))
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
		log.Printf("[ERROR] Failed to unmarshal chat message: %v", err)
		return
	}

	log.Printf("[INFO] Received chat message from SenderID: %s, RoomID: %s, Message: %s", chatMsg.SenderID, chatMsg.RoomID, chatMsg.Message)

	// 수신자가 존재하는지 확인
	if receiverConn, ok := app.ChatClients.Load(chatMsg.ReceiverID); ok {
		conn := receiverConn.(*websocket.Conn)
		log.Printf("[INFO] Sending message to ReceiverID: %s", chatMsg.ReceiverID)

		// chatMsg 객체 자체를 WriteJSON으로 전송
		if err := conn.WriteJSON(chatMsg); err != nil {
			log.Printf("[ERROR] Failed to send message to ReceiverID %s: %v", chatMsg.ReceiverID, err)
		}

		emitter, err := event.NewEventEmitter(app.Rabbit)
		if err == nil {
			log.Printf("[INFO] Pushing chat message to RabbitMQ for ReceiverID: %s", chatMsg.ReceiverID)
			emitter.PushChatMessageToQueue(event.ChatMessage(chatMsg))
		} else {
			log.Printf("[ERROR] Failed to create event emitter: %v", err)
		}
	} else {
		// 수신자가 존재하지 않는 경우
		log.Printf("[WARNING] ReceiverID %s not connected", chatMsg.ReceiverID)

		// // sync.Map에 저장된 모든 클라이언트를 출력
		// log.Println("[DEBUG] Dumping all sync.Map clients as Receiver is not connected")
		// app.ChatClients.Range(func(key, value interface{}) bool {
		// 	clientID := key.(string) // client ID (ReceiverID)
		// 	log.Printf("[CLIENTS] Connected ClientID: %s", clientID)
		// 	return true
		// })
	}
}

// Register 메시지 처리
func (app *Config) RegisterChatClient(conn *websocket.Conn, userID string) {
	app.ChatClients.Store(userID, conn)
	log.Printf("User %s register chat server", userID)
}

// UnRegister 메시지 처리
func (app *Config) UnRegisterChatClient(userID string) {
	app.ChatClients.Delete(userID)
	log.Printf("User %s unregister chat server", userID)
}
