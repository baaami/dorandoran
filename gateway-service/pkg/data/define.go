package data

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
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
	ID         string    `bson:"id" json:"id"` // UUID 사용
	Users      []string  `bson:"users" json:"users"`
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
	ModifiedAt time.Time `bson:"modified_at" json:"modified_at"`
}

type ChatLastest struct {
	RoomID string `bson:"room_id" json:"room_id"`
}
