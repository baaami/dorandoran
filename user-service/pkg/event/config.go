package event

import "encoding/json"

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

// Event Types
const (
	EventTypeChat               = "chat"
	EventTypeMatch              = "match"
	EventTypeChatLatest         = "chat.latest"
	EventTypeRoomLeave          = "room.leave"
	EventTypeRoomCreate         = "room.create"
	EventTypeCoupleRoomCreate   = "couple.room.create"
	EventTypeRoomTimeout        = "room.timeout"
	EventTypeRoomRemainTime     = "room.remain.time"
	EventTypeFinalChoiceTimeout = "final.choice.timeout"
)

// Exchange Names
const (
	ExchangeAppTopic    = "app_topic"
	ExchangeMatchEvents = "match_events"
)
