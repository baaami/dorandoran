package socket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/baaami/dorandoran/broker/event"
	"github.com/baaami/dorandoran/broker/pkg/redis"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	MessageTypeChat       = "chat"
	MessageTypeMatch      = "match"
	MessageTypeRegister   = "register"
	MessageTypeUnRegister = "unregister"
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

type UnRegisterMessage struct {
	UserID string `json:"user_id"`
}

type MatchMessage struct {
	UserID string `json:"user_id"`
}

type MatchResponse struct {
	RoomID string `json:"room_id"`
}

type ChatMessage struct {
	RoomID     string `json:"room_id"`
	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
	Message    string `json:"message"`
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
	defer conn.Close()

	// // URL에서 유저 ID 가져오기
	// userIDStr := r.Header.Get("X-User-ID")
	// userID, err := strconv.Atoi(userIDStr)
	// if err != nil {
	// 	log.Printf("Failed to Atoi user ID, err: %s", err.Error())
	// 	http.Error(w, "Failed to Atoi user ID", http.StatusInternalServerError)
	// 	return
	// }

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			// 클라이언트가 정상적으로 연결을 끊었을 경우 처리
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				log.Printf("Unexpected WebSocket close error: %v", err)
			} else {
				log.Println("WebSocket connection closed by client")
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
			app.handleRegisterMessage(conn, wsMsg.Payload)
		case MessageTypeUnRegister:
			app.handleUnRegisterMessage(conn, wsMsg.Payload)
		case MessageTypeChat:
			app.handleChatMessage(wsMsg.Payload)
		case MessageTypeMatch:
			app.handleMatchMessage(wsMsg.Payload)
		}
	}
}

// WebSocket 연결 처리
func (app *Config) HandleMatchSocket(w http.ResponseWriter, r *http.Request) {
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
	defer conn.Close()

	// // URL에서 유저 ID 가져오기
	// userIDStr := r.Header.Get("X-User-ID")
	// userID, err := strconv.Atoi(userIDStr)
	// if err != nil {
	// 	log.Printf("Failed to Atoi user ID, err: %s", err.Error())
	// 	http.Error(w, "Failed to Atoi user ID", http.StatusInternalServerError)
	// 	return
	// }

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			// 클라이언트가 정상적으로 연결을 끊었을 경우 처리
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				log.Printf("Unexpected WebSocket close error: %v", err)
			} else {
				log.Println("WebSocket connection closed by client")
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
			app.handleRegisterMessage(conn, wsMsg.Payload)
		case MessageTypeUnRegister:
			app.handleUnRegisterMessage(conn, wsMsg.Payload)
		case MessageTypeChat:
			app.handleChatMessage(wsMsg.Payload)
		case MessageTypeMatch:
			app.handleMatchMessage(wsMsg.Payload)
		}
	}
}

// Register 메시지 처리
func (app *Config) handleRegisterMessage(conn *websocket.Conn, payload json.RawMessage) {
	var regiMsg RegisterMessage
	if err := json.Unmarshal(payload, &regiMsg); err != nil {
		log.Printf("Failed to unmarshal register message: %v", err)
		return
	}

	app.Clients.Store(regiMsg.UserID, conn)
	log.Printf("User %s registered", regiMsg.UserID)
}

// UnRegister 메시지 처리
func (app *Config) handleUnRegisterMessage(conn *websocket.Conn, payload json.RawMessage) {
	var UnRegiMsg UnRegisterMessage
	if err := json.Unmarshal(payload, &UnRegiMsg); err != nil {
		log.Printf("Failed to unmarshal register message: %v", err)
		return
	}

	app.Clients.Delete(UnRegiMsg.UserID)

	conn.Close()
	log.Printf("User %s unregistered", UnRegiMsg.UserID)
}

// Chat 메시지 처리
func (app *Config) handleChatMessage(payload json.RawMessage) {
	var chatMsg ChatMessage
	if err := json.Unmarshal(payload, &chatMsg); err != nil {
		log.Printf("Failed to unmarshal chat message: %v", err)
		return
	}

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

// Match 메시지 처리
func (app *Config) handleMatchMessage(payload json.RawMessage) {
	var matchMsg MatchMessage
	if err := json.Unmarshal(payload, &matchMsg); err != nil {
		log.Printf("Failed to unmarshal match message: %v", err)
		return
	}

	app.RedisClient.AddUserToQueue(matchMsg.UserID)
	log.Printf("User %s added to waiting queue", matchMsg.UserID)
}

// Redis 대기열을 계속 확인하고 매칭 시도
func (app *Config) MonitorQueue() {
	const MatchTotalNum = 2

	for {
		matchList, err := app.RedisClient.PopNUsersFromQueue(MatchTotalNum)
		if err != nil {
			log.Printf("Error in matching: %v", err)
			continue
		}

		if len(matchList) == MatchTotalNum {
			roomID := uuid.New().String()
			log.Printf("Matched %v in room %s", matchList, roomID)
			app.notifyUsers(matchList, roomID)
		}

		time.Sleep(2 * time.Second)
	}
}

func (app *Config) notifyUsers(matchList []string, roomID string) {
	matchMsg := MatchResponse{
		RoomID: roomID,
	}

	for _, userID := range matchList {
		if conn, ok := app.Clients.Load(userID); ok {
			conn.(*websocket.Conn).WriteJSON(matchMsg)
			log.Printf("Notified %s about match in room %s", userID, roomID)
		} else {
			log.Printf("User %s not connected", userID)
		}
	}
}
