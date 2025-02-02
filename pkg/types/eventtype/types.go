package eventtypes

import (
	"encoding/json"
	"solo/pkg/types/commontype"
)

type EventPayload struct {
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
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
