package chat

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/baaami/dorandoran/broker/event"
	"github.com/baaami/dorandoran/broker/pkg/types"
	common "github.com/baaami/dorandoran/common/chat"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RoomJoinEvent struct {
	RoomID string    `bson:"room_id" json:"room_id"`
	UserID string    `bson:"user_id" json:"user_id"`
	JoinAt time.Time `bson:"join_at" json:"join_at"`
}

// BroadCast 메시지 처리
func (app *Config) handleBroadCastMessage(payload json.RawMessage, userID string) {
	var broadCastMsg ChatMessage
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
	activeUserIDs, err := app.getActiveUserIDs(broadCastMsg.RoomID)
	if err != nil {
		log.Printf("Failed to get active users for room %s: %v", broadCastMsg.RoomID, err)
		return
	}

	now := time.Now()
	chat := Chat{
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

// BroadcastToRoom handles broadcasting messages to a specific room
func (app *Config) BroadcastToRoom(chatMsg *Chat, activeUserIds []string) error {
	// WebSocket 메시지 생성
	payload, err := json.Marshal(chatMsg)
	if err != nil {
		log.Printf("Failed to marshal chatMsg: %v", err)
		return err
	}
	webSocketMsg := WebSocketMessage{
		Kind:    MessageKindMessage,
		Payload: json.RawMessage(payload),
	}

	// Room에 Websocket 메시지 전송
	if err := app.sendMessageToRoom(chatMsg.RoomID, webSocketMsg); err != nil {
		log.Printf("Failed to send message to room %s: %v", chatMsg.RoomID, err)
	}

	// RabbitMQ에 메시지 푸시
	log.Printf("Pushing chat message to RabbitMQ, room: %s", chatMsg.RoomID)
	if err := app.ChatEmitter.PushChatToQueue(event.Chat(*chatMsg)); err != nil {
		log.Printf("Failed to push chat event to queue, chatMsg: %v, err: %v", chatMsg, err)
		return err
	}

	// RabbitMQ에 활성 사용자 ID 리스트와 함께 읽음 이벤트 푸시
	readersEvent := event.ChatReadersEvent{
		MessageId: chatMsg.MessageId,
		RoomID:    chatMsg.RoomID,
		UserIds:   activeUserIds,
		ReadAt:    time.Now(),
	}
	if err := app.ChatEmitter.PushChatReadersToQueue(readersEvent); err != nil {
		log.Printf("Failed to push chat readers event: %v", err)
		return err
	}

	return nil
}

// 해당 room의 활성 사용자 수 계산
func (app *Config) getActiveUserIDs(roomID string) ([]string, error) {
	if room, ok := app.Rooms.Load(roomID); ok {
		roomMap := room.(*sync.Map)
		activeUsers := []string{}

		roomMap.Range(func(userID, clientInterface interface{}) bool {
			if client, ok := clientInterface.(*Client); ok && client != nil {
				activeUsers = append(activeUsers, userID.(string))
			}
			return true
		})
		return activeUsers, nil
	}
	return nil, fmt.Errorf("room %s not found", roomID)
}

// 해당 room에 채팅 송신
func (app *Config) sendMessageToRoom(roomID string, message WebSocketMessage) error {
	if room, ok := app.Rooms.Load(roomID); ok {
		roomMap := room.(*sync.Map)
		roomMap.Range(func(userID, clientInterface interface{}) bool {
			if client, ok := clientInterface.(*Client); ok && client != nil {
				if !app.sendMessageToClient(client, message) {
					log.Printf("Failed to send message to user %v in room %s", userID, roomID)
					roomMap.Delete(userID)
				}
			} else {
				log.Printf("Invalid client for user %v in room %s", userID, roomID)
				roomMap.Delete(userID)
			}
			return true
		})
		return nil
	}
	return fmt.Errorf("room %s not found", roomID)
}

// 해당 client에 채팅 송신
func (app *Config) sendMessageToClient(client *Client, message WebSocketMessage) bool {
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
	var joinMsg common.JoinRoomMessage
	if err := json.Unmarshal(payload, &joinMsg); err != nil {
		log.Printf("Failed to unmarshal join message: %v, err: %v", payload, err)
		return
	}

	app.JoinRoom(joinMsg.RoomID, userID)
}

// Leave 메시지 처리
func (app *Config) handleLeaveMessage(payload json.RawMessage, userID string) {
	var leaveMsg common.LeaveRoomMessage
	if err := json.Unmarshal(payload, &leaveMsg); err != nil {
		log.Printf("Failed to unmarshal leave message: %v, err: %v", payload, err)
		return
	}

	app.LeaveRoom(leaveMsg.RoomID, userID)
}

func (app *Config) JoinRoom(roomID string, userID string) {
	room, _ := app.Rooms.LoadOrStore(roomID, &sync.Map{})

	clientInterface, ok := app.ChatClients.Load(userID)
	if !ok {
		log.Printf("[ERROR] Client not found for user %s", userID)
		return
	}
	client := clientInterface.(*Client)

	room.(*sync.Map).Store(userID, client)
	log.Printf("User %s joined room %s", userID, roomID)

	roomJoinMsg := RoomJoinEvent{
		RoomID: roomID,
		UserID: userID,
		JoinAt: time.Now(),
	}

	log.Printf("Pushing room join event to RabbitMQ, roomID: %s, userID: %s, time: %v", roomJoinMsg.RoomID, roomJoinMsg.UserID, roomJoinMsg.JoinAt)

	err := app.ChatEmitter.PushRoomJoinToQueue(event.RoomJoinEvent(roomJoinMsg))
	if err != nil {
		log.Printf("Failed to push room join to queue, roomJoinMsg: %v, err: %v", roomJoinMsg, err)
	}
}

// Room에서 사용자 제거하기
func (app *Config) LeaveRoom(roomID string, userID string) {
	if room, ok := app.Rooms.Load(roomID); ok {
		room.(*sync.Map).Delete(userID) // roomID에 해당하는 사용자 제거
		log.Printf("User %s left room %s", userID, roomID)
	}
}
