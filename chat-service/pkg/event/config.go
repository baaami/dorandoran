package event

import "encoding/json"

type ExchangeConfig struct {
	Name string
	Type string
}

type RoutingConfig struct {
	Exchange ExchangeConfig
	Keys     []string
}

// EventPayload 구조체 정의
type EventPayload struct {
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

type ChatLatestEvent struct {
	RoomID string `json:"room_id"`
}

type RoomLeaveEvent struct {
	LeaveUserID int    `json:"leave_user_id"`
	RoomID      string `json:"room_id"`
}

// RoomTimeoutEvent 정의
type RoomTimeoutEvent struct {
	RoomID string `json:"room_id"`
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
	ExchangeAppTopic               = "app_topic"
	ExchangeChatRoomCreateEvents   = "chat_room_create_events"
	ExchangeCoupleRoomCreateEvents = "couple_room_create_events"
	ExchangeMatchEvents            = "match_events"
)
