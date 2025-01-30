package event

const EXCHNAGE_APP_TOPIC = "app_topic"
const EXCHNAGE_COUPLE_ROOM_CREATE = "app_topic"

type ExchangeConfig struct {
	Name string
	Type string // topic, fanout 등
}

type RoutingConfig struct {
	Exchange ExchangeConfig
	Keys     []string
}

// Event Types
const (
	EventTypeChat               = "chat"
	EventTypeChatLatest         = "chat.latest"
	EventTypeRoomLeave          = "room.leave"
	EventTypeRoomCreate         = "room.create"
	EventTypeCoupleRoomCreate   = "couple.room.create"
	EventTypeRoomTimeout        = "room.timeout"
	EventTypeFinalChoiceTimeout = "final.choice.timeout"
	EventTypeRoomRemainTime     = "room.remain.time"
)

// Exchange Names
const (
	ExchangeAppTopic               = "app_topic"
	ExchangeChatRoomCreateEvents   = "chat_room_create_events"
	ExchangeCoupleRoomCreateEvents = "couple_room_create_events"
)

type RoomLeaveEvent struct {
	LeaveUserID int    `json:"leave_user_id"`
	RoomID      string `json:"room_id"`
}

type RoomTimeoutEvent struct {
	RoomID          string `json:"room_id"`
	InactiveUserIds []int  `json:"inactive_user_ids"`
}

type FinalChoiceEvent struct {
	RoomID string `json:"room_id"`
}
