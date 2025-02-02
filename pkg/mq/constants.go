package mq

// Exchange Names
const (
	ExchangeAppTopic = "app_topic"
	ExchangeMatch    = "match"
)

// Exchange Types
const (
	ExchangeTypeTopic  = "topic"
	ExchangeTypeFanout = "fanout"
)

// Queue Names
const (
	QueueUser = "user_queue"
)

// Routing Keys
const (
	RoutingKeyFinalChoiceTimeout = "final_choice_timeout"
	RoutingKeyMatchCreated       = "match_created"
	RoutingKeyRoomLeave          = "room_leave"
)
