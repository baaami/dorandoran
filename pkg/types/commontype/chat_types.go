package commontype

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Chat struct {
	MessageId   primitive.ObjectID `bson:"_id" json:"message_id"`
	Type        string             `bson:"type" json:"type"`
	RoomID      string             `bson:"room_id" json:"room_id"`
	SenderID    int                `bson:"sender_id" json:"sender_id"`
	Message     string             `bson:"message" json:"message"`
	UnreadCount int                `bson:"unread_count" json:"unread_count"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}

type ChatReader struct {
	MessageId primitive.ObjectID `bson:"message_id" json:"message_id"`
	RoomID    string             `bson:"room_id" json:"room_id"`
	UserId    int                `bson:"user_id" json:"user_id"`
	ReadAt    time.Time          `bson:"read_at" json:"read_at"`
}
