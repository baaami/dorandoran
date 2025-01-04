package types

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatEvent struct {
	MessageId       primitive.ObjectID `bson:"_id,omitempty" json:"message_id"`
	Type            string             `bson:"type" json:"type"`
	RoomID          string             `bson:"room_id" json:"room_id"`
	SenderID        int                `bson:"sender_id" json:"sender_id"`
	Message         string             `bson:"message" json:"message"`
	UnreadCount     int                `bson:"unread_count" json:"unread_count"`
	InactiveUserIds []int              `bson:"inactive_user_ids" json:"inactive_user_ids"`
	ReaderIds       []int              `bson:"reader_ids" json:"reader_ids"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
}

// Headings 구조체
type Headings struct {
	Ko string `json:"ko"`
}

// Contents 구조체
type Contents struct {
	Ko string `json:"ko"`
}

// IncludeAliases 구조체
type IncludeAliases struct {
	ExternalID []string `json:"external_id"`
}

// PushMessage 구조체
type PushMessage struct {
	AppID          string            `json:"app_id"`
	IncludeAliases IncludeAliases    `json:"include_aliases"`
	TargetChannel  string            `json:"target_channel"`
	Headings       map[string]string `json:"headings"`
	Contents       map[string]string `json:"contents"`
	AppUrl         string            `json:"app_url"`
}
