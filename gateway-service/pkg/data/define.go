package data

import (
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	MATCHING_ROOM = iota
	DATE_ROOM
)

type Chat struct {
	MessageId   primitive.ObjectID `bson:"_id,omitempty" json:"message_id"`
	Type        string             `bson:"type" json:"type"`
	RoomID      string             `bson:"room_id" json:"room_id"`
	SenderID    int                `bson:"sender_id" json:"sender_id"`
	Message     string             `bson:"message" json:"message"`
	UnreadCount int                `bson:"unread_count" json:"unread_count"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}

type ChatMessage struct {
	HeadCnt int    `json:"head_cnt"`
	RoomID  string `json:"room_id"`
	Message string `json:"message"`
}

type ChatRoom struct {
	ID           string    `bson:"id" json:"id"` // UUID 사용
	Type         int       `bson:"type" json:"type"`
	UserIDs      []int     `bson:"user_ids" json:"user_ids"`
	CreatedAt    time.Time `bson:"created_at" json:"created_at"`
	FinishChatAt time.Time `bson:"finish_chat_at" json:"finish_chat_at"`
	ModifiedAt   time.Time `bson:"modified_at" json:"modified_at"`
}

type ChatLastest struct {
	RoomID string `bson:"room_id" json:"room_id"`
}

type ChatLatestEvent struct {
	RoomID string `json:"room_id"`
}

type WebSocketMessage struct {
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload"`
}

type RoomRemainingEvent struct {
	RoomID    string `json:"room_id"`
	Remaining int    `json:"remaining"` // 남은 시간 (초)
}

type RoomTimeoutEvent struct {
	RoomID string `json:"room_id"`
}

const (
	MessageKindMessage       = "message"
	MessageKindJoin          = "join"
	MessageKindLeave         = "leave"
	MessageKindCheckRead     = "check_read"
	MessageKindChatLastest   = "chat_latest"
	MessageKindRoomRemaining = "room_remaining"
	MessageKindRoomTimeout   = "room_timeout"
)
