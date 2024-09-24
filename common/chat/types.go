package common

import (
	"encoding/json"
	"time"
)

// Request

type WebSocketMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type RegisterMessage struct {
	UserID string `json:"user_id"`
}

type UnRegisterMessage struct {
	UserID string `json:"user_id"`
}

type MatchMessage struct {
	UserID string `json:"user_id"`
}

type ChatMessage struct {
	RoomID     string `json:"room_id"`
	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
	Message    string `json:"message"`
}

type ChatRoom struct {
	ID            int       `bson:"id" json:"id"`
	UserAID       int       `bson:"user_a_id" json:"user_a_id"`
	UserBID       int       `bson:"user_b_id" json:"user_b_id"`
	LastConfirmID int       `bson:"last_confirm_id" json:"last_confirm_id"`
	CreatedAt     time.Time `bson:"created_at" json:"created_at"`
	ModifiedAt    time.Time `bson:"modified_at" json:"modified_at"`
	ConfirmAt     time.Time `bson:"confirm_at" json:"confirm_at"`
}

// Response

type MatchResponse struct {
	RoomID string `json:"room_id"`
}
