package common

import (
	"encoding/json"
	"time"
)

// Request

type WebSocketMessage struct {
	Type    string          `json:"type"`
	Status  string          `json:"status"`
	Payload json.RawMessage `json:"payload"`
}

type RegisterMessage struct {
	UserID string `json:"user_id"`
}

type UnRegisterMessage struct {
	UserID string `json:"user_id"`
}

type JoinRoomMessage struct {
	RoomID string `json:"room_id"`
}

type LeaveRoomMessage struct {
	RoomID string `json:"room_id"`
}

type MatchMessage struct {
	UserID string `json:"user_id"`
}

type Chat struct {
	Type      string    `bson:"type" json:"type"`
	RoomID    string    `bson:"room_id" json:"room_id"`
	SenderID  int       `bson:"sender_id" json:"sender_id"`
	Message   string    `bson:"message" json:"message"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

type ChatRoom struct {
	ID         string    `bson:"id" json:"id"` // UUID 사용
	Users      []string  `bson:"users" json:"users"`
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
	ModifiedAt time.Time `bson:"modified_at" json:"modified_at"`
	// 추가적으로 각 사용자의 마지막 확인 메시지 ID를 추적하기 위한 필드를 고려할 수 있음
	UserLastRead map[string]time.Time `bson:"user_last_read" json:"user_last_read"`
}

// Push Message
type PushMatchSuccessMessage struct {
	RoomID string `json:"room_id"`
}

type PushRoomInfoMessage struct {
	ID        string    `bson:"id" json:"id"` // UUID 사용
	Users     []User    `bson:"users" json:"users"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

const (
	ChatTypeChat  = "chat"
	ChatTypeJoin  = "join"
	ChatTypeLeave = "leave"
)
