package common

import (
	"encoding/json"
	"time"

	common "github.com/baaami/dorandoran/common/user"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Request

type WebSocketMessage struct {
	Kind    string          `json:"kind"`
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
	MessageId   primitive.ObjectID `bson:"_id,omitempty" json:"message_id"`
	Type        string             `bson:"type" json:"type"`
	RoomID      string             `bson:"room_id" json:"room_id"`
	SenderID    int                `bson:"sender_id" json:"sender_id"`
	Message     string             `bson:"message" json:"message"`
	UnreadCount int                `bson:"unread_count" json:"unread_count"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}

type ChatRoom struct {
	ID         string    `bson:"id" json:"id"` // UUID 사용
	Type       int       `bson:"type" json:"type"`
	Users      []string  `bson:"users" json:"users"`
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
	ModifiedAt time.Time `bson:"modified_at" json:"modified_at"`
}

// Push Message
type PushMatchSuccessMessage struct {
	RoomID string `json:"room_id"`
}

type PushRoomInfoMessage struct {
	ID        string        `bson:"id" json:"id"` // UUID 사용
	Users     []common.User `bson:"users" json:"users"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
}

const (
	ChatTypeChat  = "chat"
	ChatTypeJoin  = "join"
	ChatTypeLeave = "leave"
)
