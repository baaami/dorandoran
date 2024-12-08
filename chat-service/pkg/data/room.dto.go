package data

import (
	"time"

	common "github.com/baaami/dorandoran/common/user"
)

type ChatRoomDetailResponse struct {
	ID         string        `bson:"id" json:"id"` // UUID 사용
	Users      []common.User `bson:"users" json:"users"`
	CreatedAt  time.Time     `bson:"created_at" json:"created_at"`
	ModifiedAt time.Time     `bson:"modified_at" json:"modified_at"`
}

type LastMessage struct {
	SenderID  int       `json:"sender_id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type ChatRoomLatestResponse struct {
	ID          string      `json:"id"`
	RoomName    string      `json:"room_name"`
	LastMessage LastMessage `json:"last_message"`
	UnreadCount int         `json:"unread_count"`
	CreatedAt   time.Time   `json:"created_at"`
	ModifiedAt  time.Time   `json:"modified_at"`
}
