package event

import (
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ExchangeConfig struct {
	Name string
	Type string // topic, fanout 등
}

type RoutingConfig struct {
	Exchange ExchangeConfig
	Keys     []string
}

type EventPayload struct {
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

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

type RoomTimeoutEvent struct {
	RoomID          string `json:"room_id"`
	InactiveUserIds []int  `json:"inactive_user_ids"`
}

const (
	EventTypeChat               = "chat"
	EventTypeRoomTimeout        = "room.timeout"
	EventTypeFinalChoiceTimeout = "final.choice.timeout"
)

const (
	ExchangeAppTopic = "app_topic"
)
