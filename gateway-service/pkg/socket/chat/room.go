package chat

import (
	"encoding/json"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/baaami/dorandoran/broker/event"
	"github.com/baaami/dorandoran/broker/pkg/types"
	common "github.com/baaami/dorandoran/common/chat"
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

	now := time.Now()
	chat := Chat{
		Type:      types.ChatTypeChat,
		RoomID:    broadCastMsg.RoomID,
		SenderID:  nUserID,
		Message:   broadCastMsg.Message,
		CreatedAt: now,
	}

	err = app.BroadcastToRoom(chat)
	if err != nil {
		log.Printf("Failed to broadcast message: %v", err)
	}
}

// Room에 있는 모든 사용자에게 브로드캐스트
func (app *Config) BroadcastToRoom(chatMsg Chat) error {
	payload, err := json.Marshal(chatMsg)
	if err != nil {
		log.Printf("Failed to marshal chatMsg: %v", err)
		return err
	}

	webSocketMsg := WebSocketMessage{
		Kind:    MessageKindMessage,
		Payload: json.RawMessage(payload),
	}

	roomID := chatMsg.RoomID
	if room, ok := app.Rooms.Load(roomID); ok {
		roomMap := room.(*sync.Map)
		roomMap.Range(func(userID, clientInterface interface{}) bool {
			if clientInterface == nil {
				log.Printf("Client for user %v in room %s is nil", userID, roomID)
				roomMap.Delete(userID)
				return true
			}

			client, ok := clientInterface.(*Client)
			if !ok || client == nil {
				log.Printf("Invalid client for user %v in room %s", userID, roomID)
				roomMap.Delete(userID)
				return true
			}

			select {
			case client.Send <- webSocketMsg:
				// 메시지 전송 성공
			case <-time.After(time.Second * 1):
				log.Printf("Time out send message to user %v in room %s", userID, roomID)
				// Optionally remove client or handle the timeout
				roomMap.Delete(userID)
			}
			return true
		})
	} else {
		log.Printf("Room %s not found", roomID)
	}

	log.Printf("Pushing chat message to RabbitMQ, room: %s", chatMsg.RoomID)

	err = app.ChatEmitter.PushChatToQueue(event.Chat(chatMsg))
	if err != nil {
		log.Printf("Failed to push chat event to queue, chatMsg: %v, err: %v", chatMsg, err)
	}

	return nil
}

func mustMarshalJSON(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		log.Fatalf("[ERROR] Failed to marshal JSON: %v", err)
	}
	return data
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

	// TODO: join time을 해당 user의 update last read 시간을 업데이트 (With. Rabbit MQ)
	// TODO: join time을 통해 room에 입장한 user들에게 check_read 소켓 이벤트를 송신 (With. WebSocket)

	log.Printf("Pushing room join event to RabbitMQ, roomID: %s, userID: %s, time: %v", roomJoinMsg.RoomID, roomJoinMsg.UserID, roomJoinMsg.JoinAt)

	err := app.ChatEmitter.PushRoomJoinToQueue(event.RoomJoinEvent(roomJoinMsg))
	if err != nil {
		log.Printf("Failed to push room join to queue, roomJoinMsg: %v, err: %v", roomJoinMsg, err)
	}

	// room에 join한 사용자에게 채팅방 정보 전송
}

// Room에서 사용자 제거하기
func (app *Config) LeaveRoom(roomID string, userID string) {
	if room, ok := app.Rooms.Load(roomID); ok {
		room.(*sync.Map).Delete(userID) // roomID에 해당하는 사용자 제거
		log.Printf("User %s left room %s", userID, roomID)
	}
}
