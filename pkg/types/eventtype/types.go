package eventtypes

import (
	"encoding/json"
	"solo/pkg/types/commontype"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EventPayload struct {
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

// Event Types
const (
	EventTypeChat               = "chat"
	EventTypeMatch              = "match"
	EventTypeChatLatest         = "chat.latest"
	EventTypeRoomLeave          = "room.leave"
	EventTypeRoomCreate         = "room.create"
	EventTypeRoomJoin           = "room.join"
	EventTypeCoupleRoomCreate   = "couple.room.create"
	EventTypeRoomTimeout        = "room.timeout"
	EventTypeRoomRemainTime     = "room.remain.time"
	EventTypeFinalChoiceTimeout = "final.choice.timeout"
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

type MatchEvent struct {
	MatchId      string                   `bson:"match_id" json:"match_id"`
	MatchType    int                      `bson:"match_type" json:"match_type"`
	MatchedUsers []commontype.WaitingUser `bson:"matched_users" json:"matched_users"`
}

type RoomLeaveEvent struct {
	LeaveUserID int    `json:"leave_user_id"`
	RoomID      string `json:"room_id"`
}

type RoomTimeoutEvent struct {
	RoomID          string `json:"room_id"`
	InactiveUserIds []int  `json:"inactive_user_ids"`
}

type FinalChoiceTimeoutEvent struct {
	RoomID  string `bson:"room_id" json:"room_id"`
	UserIDs []int  `bson:"user_ids" json:"user_ids"`
}

type RoomJoinEvent struct {
	RoomID string    `bson:"room_id" json:"room_id"`
	UserID int       `bson:"user_id" json:"user_id"`
	JoinAt time.Time `bson:"join_at" json:"join_at"`
}

type ChatLatestEvent struct {
	RoomID string `json:"room_id"`
}

type FinalChoiceEvent struct {
	RoomID string `json:"room_id"`
}
