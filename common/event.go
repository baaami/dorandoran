package common

import "encoding/json"

type EventPayload struct {
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

type RoomJoinEvent struct {
	RoomID string `bson:"room_id" json:"room_id"`
	UserID string `bson:"user_id" json:"user_id"`
}
