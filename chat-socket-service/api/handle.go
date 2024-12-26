package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/baaami/dorandoran/chat-socket-service/pkg/types"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Client struct {
	Conn *websocket.Conn
	Send chan interface{}
}

func (app *Config) HandleChatSocket(c echo.Context) error {
	// 컨텍스트 생성 및 취소 함수 정의
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "WebSocket upgrade failed")
	}
	defer conn.Close()

	userID := c.Request().Header.Get("X-User-ID")

	app.RegisterChatClient(conn, userID)
	defer func() {
		app.UnRegisterChatClient(userID)
		conn.Close()
	}()

	// WaitGroup을 사용하여 모든 고루틴이 종료될 때까지 대기
	var wg sync.WaitGroup
	wg.Add(1) // 두 개의 고루틴 (listenChatEvent, pingPump)

	// 게임에 필요한 초기 정보 전달

	// 메시지 처리 고루틴
	go func() {
		defer wg.Done()
		app.listenChatEvent(ctx, conn, userID)
	}()

	// 모든 고루틴이 종료될 때까지 대기
	wg.Wait()

	return nil
}

// 메시지 읽기 처리
func (app *Config) listenChatEvent(ctx context.Context, conn *websocket.Conn, userID string) {
	for {
		select {
		case <-ctx.Done():
			return // 컨텍스트가 취소되면 고루틴 종료
		default:
			_, msg, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
					log.Printf("Unexpected WebSocket close error")
				} else {
					log.Printf("WebSocket connection closed by client")
				}
				return
			}

			var wsMsg types.WebSocketMessage
			if err := json.Unmarshal(msg, &wsMsg); err != nil {
				log.Printf("Failed to unmarshal message: %v", err)
				continue
			}

			switch wsMsg.Kind {
			case types.MessageKindMessage:
				app.handleBroadCastMessage(wsMsg.Payload, userID)
			case types.MessageKindJoin:
				app.handleJoinMessage(wsMsg.Payload, userID)
			case types.MessageKindLeave:
				app.handleLeaveMessage(wsMsg.Payload, userID)
			}
		}
	}
}

func (app *Config) handleBroadCastMessage(payload json.RawMessage, userID string) {
	var broadCastMsg types.ChatMessage
	if err := json.Unmarshal(payload, &broadCastMsg); err != nil {
		log.Printf("Failed to unmarshal broadcast message: %v", err)
		return
	}

	nUserID, err := strconv.Atoi(userID)
	if err != nil {
		log.Printf("Failed to atoi, userid: %s, err: %s", userID, err.Error())
		return
	}

	// 활성 사용자 ID 리스트 가져오기
	activeUserIDs, err := app.RedisClient.GetActiveUserIDs(broadCastMsg.RoomID)
	if err != nil {
		log.Printf("Failed to get active users for room %s: %v", broadCastMsg.RoomID, err)
		return
	}

	now := time.Now()
	chat := types.Chat{
		MessageId:   primitive.NewObjectID(),
		Type:        types.ChatTypeChat,
		RoomID:      broadCastMsg.RoomID,
		SenderID:    nUserID,
		Message:     broadCastMsg.Message,
		UnreadCount: broadCastMsg.HeadCnt - len(activeUserIDs), // 활성 사용자 수를 이용해 UnreadCount 계산
		CreatedAt:   now,
	}

	// Broadcast to the room
	if err := app.BroadcastToRoom(&chat, activeUserIDs); err != nil {
		log.Printf("Failed to broadcast message: %v", err)
	}
}

func (app *Config) BroadcastToRoom(chatMsg *types.Chat, activeUserIds []string) error {
	// WebSocket 메시지 생성
	payload, err := json.Marshal(chatMsg)
	if err != nil {
		log.Printf("Failed to marshal chatMsg: %v", err)
		return err
	}
	webSocketMsg := types.WebSocketMessage{
		Kind:    types.MessageKindMessage,
		Payload: json.RawMessage(payload),
	}

	// Room에 Websocket 메시지 전송
	if err := app.sendMessageToRoom(chatMsg.RoomID, webSocketMsg); err != nil {
		log.Printf("Failed to send message to room %s: %v", chatMsg.RoomID, err)
	}

	chatEvent := types.ChatEvent{
		MessageId:   chatMsg.MessageId,
		Type:        chatMsg.Type,
		RoomID:      chatMsg.RoomID,
		SenderID:    chatMsg.SenderID,
		Message:     chatMsg.Message,
		UnreadCount: chatMsg.UnreadCount,
		ReaderIds:   activeUserIds,
		CreatedAt:   chatMsg.CreatedAt,
	}

	// RabbitMQ에 메시지 푸시
	log.Printf("Pushing chat message to RabbitMQ, room: %s, active: %v", chatMsg.RoomID, activeUserIds)
	if err := app.ChatEmitter.PushChatToQueue(chatEvent); err != nil {
		log.Printf("Failed to push chat event to queue, chatMsg: %v, err: %v", chatMsg, err)
		return err
	}

	return nil
}

func (app *Config) sendMessageToRoom(roomID string, message types.WebSocketMessage) error {
	activeUserIDs, err := app.RedisClient.GetActiveUserIDs(roomID)
	if err != nil {
		log.Printf("Failed to get active user id list, err: %s", err.Error())
		return err
	}

	log.Printf("[sendMessageToRoom] active user: %v", activeUserIDs)

	for _, activeUserID := range activeUserIDs {
		if client, ok := app.ChatClients.Load(activeUserID); ok {
			if !app.sendMessageToClient(client.(*Client), message) {
				log.Printf("Failed to send message to user %v in room %s", activeUserID, roomID)
			}
		}
	}

	return nil
}

func (app *Config) sendMessageToClient(client *Client, message types.WebSocketMessage) bool {
	select {
	case client.Send <- message:
		return true // 메시지 전송 성공
	case <-time.After(time.Second * 1):
		log.Printf("Timeout while sending message")
		return false // 메시지 전송 실패
	}
}

// Join 메시지 처리
func (app *Config) handleJoinMessage(payload json.RawMessage, userID string) {
	var joinMsg types.JoinRoomMessage
	if err := json.Unmarshal(payload, &joinMsg); err != nil {
		log.Printf("Failed to unmarshal join message: %v, err: %v", payload, err)
		return
	}

	app.JoinRoom(joinMsg.RoomID, userID)
}

// Leave 메시지 처리
func (app *Config) handleLeaveMessage(payload json.RawMessage, userID string) {
	var leaveMsg types.LeaveRoomMessage
	if err := json.Unmarshal(payload, &leaveMsg); err != nil {
		log.Printf("Failed to unmarshal leave message: %v, err: %v", payload, err)
		return
	}
}

func (app *Config) JoinRoom(roomID string, userID string) {
	log.Printf("User %s joined room %s", userID, roomID)

	roomJoinMsg := types.RoomJoinEvent{
		RoomID: roomID,
		UserID: userID,
		JoinAt: time.Now(),
	}

	log.Printf("Pushing room join event to RabbitMQ, roomID: %s, userID: %s, time: %v", roomJoinMsg.RoomID, roomJoinMsg.UserID, roomJoinMsg.JoinAt)

	err := app.ChatEmitter.PushRoomJoinToQueue(types.RoomJoinEvent(roomJoinMsg))
	if err != nil {
		log.Printf("Failed to push room join to queue, roomJoinMsg: %v, err: %v", roomJoinMsg, err)
	}
}

func (app *Config) RegisterChatClient(conn *websocket.Conn, userID string) {
	client := &Client{
		Conn: conn,
		Send: make(chan interface{}, 256),
	}

	// 쓰기 고루틴 시작
	go client.writePump()

	app.ChatClients.Store(userID, client)

	// Redis에 활성 사용자로 등록
	serverID := "unique-server-id" // TODO: 서버의 고유 식별자
	if err := app.RedisClient.RegisterActiveUser(userID, serverID); err != nil {
		log.Printf("Failed to register active user %s in Redis: %v", userID, err)
	} else {
		log.Printf("User %s registered as active on server %s", userID, serverID)
	}

	log.Printf("User %s register chat server", userID)
}

func (app *Config) UnRegisterChatClient(userID string) {
	if clientInterface, ok := app.ChatClients.Load(userID); ok {
		client := clientInterface.(*Client)

		// Send 채널 닫기
		close(client.Send)

		// Channel에서 유저 제거
		app.ChatClients.Delete(userID)

		// Redis에서 활성 사용자 제거
		if err := app.RedisClient.UnregisterActiveUser(userID); err != nil {
			log.Printf("Failed to unregister active user %s in Redis: %v", userID, err)
		} else {
			log.Printf("User %s unregistered as active", userID)
		}

		log.Printf("User %s unregistered chat server", userID)
	}
}

func (c *Client) writePump() {
	defer func() {
		c.Conn.Close()
		log.Printf("[INFO] writePump for user %v exited", c.Conn.RemoteAddr())
	}()

	for {
		message, ok := <-c.Send
		if !ok {
			// 채널이 닫힌 경우 연결을 닫습니다.
			c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		// 메시지를 전송합니다.
		if err := c.Conn.WriteJSON(message); err != nil {
			log.Printf("Failed to write message: %v", err)
			return
		}
	}
}

func (app *Config) SendSocketByChatEvents() {
	for event := range app.EventChannel {
		switch event.Kind {
		case types.MessageKindChatLastest:
			if err := app.handleChatLatestEvent(event.Payload); err != nil {
				log.Printf("Failed to handle chat.latest event: %v", err)
			}
		case types.MessageKindMessage:
			if err := app.handleChatEvent(event.Payload); err != nil {
				log.Printf("Failed to handle chat event: %v", err)
			}
		default:
			log.Printf("Unknown WebSocket event kind: %s", event.Kind)
		}
	}
}

func (app *Config) handleChatLatestEvent(payload json.RawMessage) error {
	var chatLatest types.ChatLatestEvent
	if err := json.Unmarshal(payload, &chatLatest); err != nil {
		return fmt.Errorf("failed to unmarshal chat.latest payload: %w", err)
	}

	log.Printf("Broadcasting chat.latest event for RoomID: %s", chatLatest.RoomID)

	wsMessage := types.WebSocketMessage{
		Kind:    types.MessageKindChatLastest,
		Payload: payload,
	}

	if err := app.sendMessageToRoom(chatLatest.RoomID, wsMessage); err != nil {
		return fmt.Errorf("failed to broadcast chat.latest for RoomID %s: %w", chatLatest.RoomID, err)
	}

	return nil
}

func (app *Config) handleChatEvent(payload json.RawMessage) error {
	var chatMsg types.ChatEvent
	if err := json.Unmarshal(payload, &chatMsg); err != nil {
		return fmt.Errorf("failed to unmarshal chat payload: %w", err)
	}

	log.Printf("Broadcasting chat event for RoomID: %s", chatMsg.RoomID)

	wsMessage := types.WebSocketMessage{
		Kind:    types.MessageKindMessage,
		Payload: payload,
	}

	if err := app.sendMessageToRoom(chatMsg.RoomID, wsMessage); err != nil {
		return fmt.Errorf("failed to broadcast chat.latest for RoomID %s: %w", chatMsg.RoomID, err)
	}

	return nil
}
