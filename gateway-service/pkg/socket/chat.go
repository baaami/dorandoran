package socket

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/baaami/dorandoran/broker/event"
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

	// 연결 성공
	defer conn.Close()

	// URL에서 유저 ID 가져오기
	userID := r.Header.Get("X-User-ID")

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
		case MessageTypeBroadCast:
			app.handleBroadCastMessage(wsMsg.Payload)
		case MessageTypeJoin:
			app.handleJoinMessage(wsMsg.Payload, userID, conn)
		case MessageTypeLeave:
			app.handleLeaveMessage(wsMsg.Payload, userID)
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
			// TODO: 재시도 로직이나 대체 방안을 고려
			emitter.PushChatMessageToQueue(event.ChatMessage(chatMsg))
		} else {
			log.Printf("[ERROR] Failed to create event emitter: %v", err)
		}
	} else {
		// 수신자가 존재하지 않는 경우
		log.Printf("[WARNING] ReceiverID %s not connected", chatMsg.ReceiverID)
	}
}

// BroadCast 메시지 처리
func (app *Config) handleBroadCastMessage(payload json.RawMessage) {
	var broadCastMsg common.ChatMessage
	if err := json.Unmarshal(payload, &broadCastMsg); err != nil {
		log.Printf("[ERROR] Failed to unmarshal join message: %v", err)
		return
	}

	app.BroadcastToRoom(broadCastMsg)
}

// Join 메시지 처리
func (app *Config) handleJoinMessage(payload json.RawMessage, userID string, conn *websocket.Conn) {
	var joinMsg JoinRoomMessage
	if err := json.Unmarshal(payload, &joinMsg); err != nil {
		log.Printf("[ERROR] Failed to unmarshal join message: %v", err)
		return
	}

	app.JoinRoom(joinMsg.RoomID, userID, conn)
}

// Leave 메시지 처리
func (app *Config) handleLeaveMessage(payload json.RawMessage, userID string) {
	var leaveMsg LeaveRoomMessage
	if err := json.Unmarshal(payload, &leaveMsg); err != nil {
		log.Printf("[ERROR] Failed to unmarshal leave message: %v", err)
		return
	}

	app.LeaveRoom(leaveMsg.RoomID, userID)
}

// Register
func (app *Config) RegisterChatClient(conn *websocket.Conn, userID string) {
	app.ChatClients.Store(userID, conn)
	log.Printf("User %s register chat server", userID)
}

// UnRegister
func (app *Config) UnRegisterChatClient(userID string) {
	// TODO: Room에서도 LeaveRoom을 해야되지 않나??
	app.ChatClients.Delete(userID)
	log.Printf("User %s unregister chat server", userID)
}
