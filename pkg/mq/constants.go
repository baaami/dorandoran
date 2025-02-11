package mq

// Exchange Names
const (
	ExchangeAppTopic             = "app_topic"
	ExchangeMatchEvents          = "match_events"
	ExchangeMatch                = "match"
	ExchangeChatRoomCreateEvents = "chat_room_create_events"
)

// Exchange Types
const (
	ExchangeTypeTopic  = "topic"
	ExchangeTypeFanout = "fanout"
)

// Queue Names
const (
	QueueUser  = "user_queue"
	QueueMatch = "match_queue"
)

// Routing Keys
const (
	RoutingKeyChat               = "chat"
	RoutingKeyFinalChoiceTimeout = "final_choice_timeout"
	RoutingKeyMatchCreated       = "match_created"
	RoutingKeyRoomLeave          = "room_leave"
	RoutingKeyRoomTimeout        = "room.timeout"
)

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
