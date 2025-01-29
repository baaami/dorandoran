package data

import (
	"time"

	"github.com/baaami/dorandoran/chat/pkg/types"
)

type RoomDetailResponse struct {
	ID                  string        `bson:"id" json:"id"` // UUID 사용
	Type                int           `bson:"type" json:"type"`
	Status              int           `bson:"status" json:"status"`
	Seq                 int           `bson:"seq" json:"seq"`
	RoomName            string        `bson:"room_name" json:"room_name"`
	Users               []types.Gamer `bson:"users" json:"users"`
	CreatedAt           time.Time     `bson:"created_at" json:"created_at"`
	FinishChatAt        time.Time     `bson:"finish_chat_at" json:"finish_chat_at"`
	FinishFinalChoiceAt time.Time     `bson:"finish_final_choice_at" json:"finish_final_choice_at"`
	ModifiedAt          time.Time     `bson:"modified_at" json:"modified_at"`
}

type RoomCharacterNameResponse struct {
	name string `json:"name"`
}

type LastMessage struct {
	SenderID  int            `json:"sender_id"`
	Message   string         `json:"message"`
	GameInfo  types.GameInfo `json:"game_info"`
	CreatedAt time.Time      `json:"created_at"`
}

type ChatRoomLatestResponse struct {
	ID          string      `json:"id"`
	RoomName    string      `json:"room_name"`
	RoomType    int         `json:"room_type"`
	LastMessage LastMessage `json:"last_message"`
	UnreadCount int         `json:"unread_count"`
	CreatedAt   time.Time   `json:"created_at"`
	ModifiedAt  time.Time   `json:"modified_at"`
}
