package sock

import "encoding/json"

const (
	PushMessageStatusMatchSuccess = "success"
	PushMessageStatusMatchFailure = "fail"
)

const (
	MessageTypeMatch = "match"
)

type WebSocketMessage struct {
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload"`
}
