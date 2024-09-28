package socket

import (
	"log"
	"sync"

	common "github.com/baaami/dorandoran/common/chat"
	"github.com/gorilla/websocket"
)

// Room에 있는 모든 사용자에게 브로드캐스트
func (app *Config) BroadcastToRoom(chatMsg common.ChatMessage) {
	roomID := chatMsg.RoomID
	if room, ok := app.Rooms.Load(roomID); ok {
		roomMap := room.(*sync.Map)
		roomMap.Range(func(userID, conn interface{}) bool {
			// conn이 nil인지 확인
			if conn == nil {
				log.Printf("[ERROR] Connection for user %v in room %s is nil", userID, roomID)
				// nil 연결을 room map에서 제거
				roomMap.Delete(userID)
				return true
			}

			wsConn, ok := conn.(*websocket.Conn)
			if !ok {
				log.Printf("[ERROR] Invalid connection type for user %v in room %s", userID, roomID)
				// 잘못된 연결을 room map에서 제거
				roomMap.Delete(userID)
				return true
			}

			// 연결이 닫혀있는지 확인
			if wsConn == nil || wsConn.CloseHandler() == nil {
				log.Printf("[INFO] Connection for user %v in room %s is closed", userID, roomID)
				// 닫힌 연결을 room map에서 제거
				roomMap.Delete(userID)
				return true
			}

			// 메시지 전송 시도
			if err := wsConn.WriteJSON(chatMsg); err != nil {
				log.Printf("[ERROR] Failed to send message to user %v in room %s: %v", userID, roomID, err)

				// 에러가 연결 종료와 관련된 경우
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
					log.Printf("[INFO] Removing closed connection for user %v in room %s", userID, roomID)
					// 닫힌 연결을 room map에서 제거
					roomMap.Delete(userID)
				} else {
					log.Printf("[INFO] Connection error for user %v in room %s: %v", userID, roomID, err)
				}
			}
			return true
		})
	} else {
		log.Printf("[WARNING] Room %s not found", roomID)
	}
}

// Room에 사용자 추가하기
func (app *Config) JoinRoom(roomID string, userID string, conn *websocket.Conn) {
	room, _ := app.Rooms.LoadOrStore(roomID, &sync.Map{}) // 각 roomID에 대해 새로운 sync.Map 생성
	room.(*sync.Map).Store(userID, conn)                  // roomID에 해당하는 사용자 저장
	log.Printf("User %s joined room %s", userID, roomID)
}

// Room에서 사용자 제거하기
func (app *Config) LeaveRoom(roomID string, userID string) {
	if room, ok := app.Rooms.Load(roomID); ok {
		room.(*sync.Map).Delete(userID) // roomID에 해당하는 사용자 제거
		log.Printf("User %s left room %s", userID, roomID)
	}
}
