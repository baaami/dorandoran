package event

import "encoding/json"

type ExchangeConfig struct {
	Name string
	Type string
}

type EventPayload struct {
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}
