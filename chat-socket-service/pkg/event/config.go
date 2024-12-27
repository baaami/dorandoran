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
	EventTypeChat             = "chat"
	EventTypeChatLatest       = "chat.latest"
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
