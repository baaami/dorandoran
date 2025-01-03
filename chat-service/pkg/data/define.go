package data

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const PAGE_DEFAULT_SIZE = 20

const (
	MATCHING_ROOM = iota
	DATE_ROOM
)

type Chat struct {
	MessageId   primitive.ObjectID `bson:"_id" json:"message_id"`
	Type        string             `bson:"type" json:"type"`
	RoomID      string             `bson:"room_id" json:"room_id"`
	SenderID    int                `bson:"sender_id" json:"sender_id"`
	Message     string             `bson:"message" json:"message"`
	UnreadCount int                `bson:"unread_count" json:"unread_count"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}

type ChatReader struct {
	MessageId primitive.ObjectID `bson:"message_id" json:"message_id"`
	RoomID    string             `bson:"room_id" json:"room_id"`
	UserId    int                `bson:"user_id" json:"user_id"`
	ReadAt    time.Time          `bson:"read_at" json:"read_at"`
}

type ChatRoom struct {
	ID           string    `bson:"id" json:"id"` // UUID 사용
	Type         int       `bson:"type" json:"type"`
	UserIDs      []int     `bson:"user_ids" json:"user_ids"`
	CreatedAt    time.Time `bson:"created_at" json:"created_at"`
	FinishChatAt time.Time `bson:"finish_chat_at" json:"finish_chat_at"`
	ModifiedAt   time.Time `bson:"modified_at" json:"modified_at"`
}

type ChatListResponse struct {
	Data        []*Chat `json:"data"`
	CurrentPage int     `json:"currentPage"`
	NextPage    int     `json:"nextPage,omitempty"`
	HasNextPage bool    `json:"hasNextPage"`
	TotalPages  int     `json:"totalPages"`
}

type ChatEvent struct {
	MessageId   primitive.ObjectID `bson:"_id,omitempty" json:"message_id"`
	Type        string             `bson:"type" json:"type"`
	RoomID      string             `bson:"room_id" json:"room_id"`
	SenderID    int                `bson:"sender_id" json:"sender_id"`
	Message     string             `bson:"message" json:"message"`
	UnreadCount int                `bson:"unread_count" json:"unread_count"`
	ReaderIds   []int              `bson:"reader_ids" json:"reader_ids"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}

type ChatReadersEvent struct {
	MessageId primitive.ObjectID `bson:"message_id" json:"message_id"`
	RoomID    string             `bson:"room_id" json:"room_id"`
	UserIds   []string           `bson:"user_ids" json:"user_ids"`
	ReadAt    time.Time          `bson:"read_at" json:"read_at"`
}

type RoomJoinEvent struct {
	RoomID string    `bson:"room_id" json:"room_id"`
	UserID int       `bson:"user_id" json:"user_id"`
	JoinAt time.Time `bson:"join_at" json:"join_at"`
}

type RoomRemainingEvent struct {
	RoomID    string `json:"room_id"`
	Remaining int    `json:"remaining"` // 남은 시간 (초)
}
