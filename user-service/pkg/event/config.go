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
