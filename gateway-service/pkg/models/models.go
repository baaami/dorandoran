package models

type ChatMessage struct {
	RoomID     string `json:"room_id"`
	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
	Message    string `json:"message"`
}