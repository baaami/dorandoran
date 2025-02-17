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
