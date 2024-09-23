package common

type RegisterMessage struct {
	UserID string `json:"user_id"`
}

type UnRegisterMessage struct {
	UserID string `json:"user_id"`
}

type MatchMessage struct {
	UserID string `json:"user_id"`
}

type MatchResponse struct {
	RoomID string `json:"room_id"`
}
