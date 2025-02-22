package dto

import (
	"solo/pkg/types/commontype"
	"time"
)

type RoomDetailResponse struct {
	ID                  string             `bson:"id" json:"id"` // UUID 사용
	Type                int                `bson:"type" json:"type"`
	Status              int                `bson:"status" json:"status"`
	Seq                 int                `bson:"seq" json:"seq"`
	RoomName            string             `bson:"room_name" json:"room_name"`
	Users               []commontype.Gamer `bson:"users" json:"users"`
	CreatedAt           time.Time          `bson:"created_at" json:"created_at"`
	FinishChatAt        time.Time          `bson:"finish_chat_at" json:"finish_chat_at"`
	FinishFinalChoiceAt time.Time          `bson:"finish_final_choice_at" json:"finish_final_choice_at"`
	ModifiedAt          time.Time          `bson:"modified_at" json:"modified_at"`
}

type RoomListResponse struct {
	ID          string      `json:"id"`
	RoomName    string      `json:"room_name"`
	RoomType    int         `json:"room_type"`
	LastMessage LastMessage `json:"last_message"`
	UnreadCount int         `json:"unread_count"`
	CreatedAt   time.Time   `json:"created_at"`
	ModifiedAt  time.Time   `json:"modified_at"`
}

type ChatListResponse struct {
	Data        []*commontype.Chat `json:"data"`
	CurrentPage int                `json:"currentPage"`
	NextPage    int                `json:"nextPage,omitempty"`
	HasNextPage bool               `json:"hasNextPage"`
	TotalPages  int                `json:"totalPages"`
}

type LastMessage struct {
	SenderID  int                 `json:"sender_id"`
	Message   string              `json:"message"`
	GameInfo  commontype.GameInfo `json:"game_info"`
	CreatedAt time.Time           `json:"created_at"`
}
