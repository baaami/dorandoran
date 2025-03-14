package stype

import "encoding/json"

type WebSocketMessage struct {
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload"`
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
	MessageKindFinalChoiceTimeout = "final_choice_timeout"
	MessageKindFinalChoiceStart   = "final_choice_start"
	MessageKindFinalChoice        = "final_choice"
	MessageKindFinalChoiceResult  = "final_choice_result"
	MessageKindCoupleMatchSuccess = "couple_match_success"
)

const (
	PushMessageStatusMatchSuccess = "success"
	PushMessageStatusMatchFailure = "fail"
)

const (
	MessageTypeMatch = "match"
)

type ChatMessage struct {
	HeadCnt int    `json:"head_cnt"`
	RoomID  string `json:"room_id"`
	Message string `json:"message"`
}

type FinalChoiceResultMessage struct {
	RoomID  string       `json:"room_id"`
	Choices []UserChoice `json:"choices"`
}

type FinalChoiceStartMessage struct {
	RoomID   string `json:"room_id"`
	RoomName string `json:"room_name"`
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

type UserChoice struct {
	UserID         int `json:"user_id"`
	SelectedUserID int `json:"selected_user_id"`
}

type CoupleMatchSuccessMessage struct {
	RoomID string `json:"room_id"`
}
