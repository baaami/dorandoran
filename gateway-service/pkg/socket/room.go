package socket

import (
	"log"
	"sync"
	"time"

	"github.com/baaami/dorandoran/broker/event"
)

type RoomJoinEvent struct {
	RoomID string `bson:"room_id" json:"room_id"`
	UserID string `bson:"user_id" json:"user_id"`
}

// Room에 있는 모든 사용자에게 브로드캐스트
func (app *Config) BroadcastToRoom(chatMsg Chat) {
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

			// 메시지를 Send 채널에 보냅니다.
			select {
			case client.Send <- chatMsg:
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

	emitter, err := event.NewEventEmitter(app.Rabbit)
	if err == nil {
		log.Printf("[INFO] Pushing chat message to RabbitMQ, room: %s", chatMsg.RoomID)

		err = emitter.PushChatToQueue(event.Chat(chatMsg))
		if err != nil {
			log.Printf("Failed to push chat event to queue, chatMsg: %v, err: %v", chatMsg, err)
		}
	} else {
		log.Printf("[ERROR] Failed to create event emitter: %v", err)
	}
}

// Room에 사용자 추가하기
func (app *Config) JoinRoom(roomID string, userID string) {
	room, _ := app.Rooms.LoadOrStore(roomID, &sync.Map{}) // 각 roomID에 대해 새로운 sync.Map 생성

	// 클라이언트 가져오기
	clientInterface, ok := app.ChatClients.Load(userID)
	if !ok {
		log.Printf("[ERROR] Client not found for user %s", userID)
		return
	}
	client := clientInterface.(*Client)

	room.(*sync.Map).Store(userID, client) // roomID에 해당하는 클라이언트 저장
	log.Printf("User %s joined room %s", userID, roomID)

	roomJoinMsg := RoomJoinEvent{
		RoomID: roomID,
		UserID: userID,
	}

	emitter, err := event.NewEventEmitter(app.Rabbit)
	if err == nil {
		log.Printf("Pushing room join event to RabbitMQ, roomID: %s, userID: %s", roomJoinMsg.RoomID, roomJoinMsg.UserID)

		err = emitter.PushRoomJoinToQueue(event.RoomJoinEvent(roomJoinMsg))
		if err != nil {
			log.Printf("Failed to push room join to queue, roomJoinMsg: %v, err: %v", roomJoinMsg, err)
		}
	} else {
		log.Printf("Failed to create event emitter: %v", err)
	}
}

// Room에서 사용자 제거하기
func (app *Config) LeaveRoom(roomID string, userID string) {
	if room, ok := app.Rooms.Load(roomID); ok {
		room.(*sync.Map).Delete(userID) // roomID에 해당하는 사용자 제거
		log.Printf("User %s left room %s", userID, roomID)
	}
}
