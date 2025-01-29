package types

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	MATCH_GAME = iota
	MATCH_COUPLE
)

const (
	USER_STATUS_STANDBY = iota
	USER_STATUS_GAME_ING
)

const ONE_GAME_POINT = 5

type Address struct {
	City     string `gorm:"size:100" json:"city"`
	District string `gorm:"size:100" json:"district"`
	Street   string `gorm:"size:100" json:"street"`
}

type WaitingUser struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Gender      int     `json:"gender"`
	Birth       string  `json:"birth"`
	Address     Address `json:"address"`
	CoupleCount int     `json:"couple_count"`
}

type MatchEvent struct {
	MatchId      string        `bson:"match_id" json:"match_id"`
	MatchType    int           `bson:"match_type" json:"match_type"`
	MatchedUsers []WaitingUser `bson:"matched_users" json:"matched_users"`
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
