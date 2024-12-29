package event

import "encoding/json"

type ExchangeConfig struct {
	Name string
	Type string
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

// Event Types
const (
	EventTypeChat             = "chat"
	EventTypeChatLatest       = "chat.latest"
	EventTypeRoomLeave        = "room.leave"
	EventTypeRoomCreate       = "room.create"
	EventTypeCoupleRoomCreate = "couple.room.create"
	EventTypeRoomTimeout      = "room.timeout"
	EventTypeRoomRemainTime   = "room.remain.time"
)

// Exchange Names
const (
	ExchangeAppTopic               = "app_topic"
	ExchangeChatRoomCreateEvents   = "chat_room_create_events"
	ExchangeCoupleRoomCreateEvents = "couple_room_create_events"
)
