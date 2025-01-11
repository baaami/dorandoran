package types

import (
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	MALE = iota
	FEMALE
)

const (
	MATCH_GAME = iota
	MATCH_COUPLE
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
	ID           string      `bson:"id" json:"id"` // UUID 사용
	Type         int         `bson:"type" json:"type"`
	UserIDs      []int       `bson:"user_ids" json:"user_ids"`
	Gamers       []GamerInfo `bson:"gamers" json:"gamers"` // 사용자별 캐릭터 정보
	Seq          int64       `bson:"seq" json:"seq"`       // 자동 증가 필드
	CreatedAt    time.Time   `bson:"created_at" json:"created_at"`
	FinishChatAt time.Time   `bson:"finish_chat_at" json:"finish_chat_at"`
	ModifiedAt   time.Time   `bson:"modified_at" json:"modified_at"`
}

type GamerInfo struct {
	UserID             int    `bson:"user_id" json:"user_id"`                             // 사용자 ID
	CharacterID        int    `bson:"character_id" json:"character_id"`                   // 캐릭터 식별자 (0 ~ 5)
	CharacterName      string `bson:"character_avatar_name" json:"character_avatar_name"` // 캐릭터 이름
	CharacterAvatarURL string `bson:"character_avatar_url" json:"character_avatar_url"`   // 캐릭터 아바타 이미지 URL
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

const (
	MessageKindPing               = "ping"
	MessageKindPong               = "pong"
	MessageKindMessage            = "message"
	MessageKindJoin               = "join"
	MessageKindLeave              = "leave"
	MessageKindCheckRead          = "check_read"
	MessageKindChatLastest        = "chat_latest"
	MessageKindRoomRemaining      = "room_remaining"
	MessageKindRoomTimeout        = "room_timeout"
	MessageKindFinalChoiceStart   = "final_choice_start"
	MessageKindFinalChoice        = "final_choice"
	MessageKindFinalChoiceResult  = "final_choice_result"
	MessageKindCoupleMatchSuccess = "couple_match_success"
)

const (
	ChatTypeChat  = "chat"
	ChatTypeJoin  = "join"
	ChatTypeLeave = "leave"
)

type UserChoice struct {
	UserID         int `json:"user_id"`
	SelectedUserID int `json:"selected_user_id"`
}

type FinalChoiceResultMessage struct {
	RoomID  string       `json:"room_id"`
	Choices []UserChoice `json:"choices"`
}

type JoinRoomMessage struct {
	RoomID string `json:"room_id"`
}

type LeaveRoomMessage struct {
	RoomID string `json:"room_id"`
}

type RoomTimeoutMessage struct {
	RoomID string `json:"room_id"`
}

type FinalChoiceMessage struct {
	RoomID         string `json:"room_id"`
	SelectedUserID int    `json:"selected_user_id"`
}

type CoupleMatchSuccessMessage struct {
	RoomID string `json:"room_id"`
}

type RoomJoinEvent struct {
	RoomID string    `bson:"room_id" json:"room_id"`
	UserID int       `bson:"user_id" json:"user_id"`
	JoinAt time.Time `bson:"join_at" json:"join_at"`
}

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

type User struct {
	ID      int     `gorm:"primaryKey;autoIncrement" json:"id"`
	SnsType int     `gorm:"index" json:"sns_type"`
	SnsID   string  `gorm:"index" json:"sns_id"`
	Name    string  `gorm:"size:100" json:"name"`
	Gender  int     `json:"gender"`
	Birth   string  `gorm:"size:20" json:"birth"`
	Address Address `gorm:"embedded;embeddedPrefix:address_" json:"address"`
}

type Couple struct {
	UserID1 int `json:"user_id_1"`
	UserID2 int `json:"user_id_2"`
}
