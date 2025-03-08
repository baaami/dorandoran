package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
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
	ID                  string      `bson:"id" json:"id"` // UUID 사용
	Name                string      `bson:"name" json:"name"`
	Type                int         `bson:"type" json:"type"`
	Status              int         `bson:"status" json:"status"`
	UserIDs             []int       `bson:"user_ids" json:"user_ids"`
	Gamers              []GamerInfo `bson:"gamers" json:"gamers"` // 사용자별 캐릭터 정보
	Seq                 int64       `bson:"seq" json:"seq"`       // 자동 증가 필드
	CreatedAt           time.Time   `bson:"created_at" json:"created_at"`
	FinishChatAt        time.Time   `bson:"finish_chat_at" json:"finish_chat_at"`
	FinishFinalChoiceAt time.Time   `bson:"finish_final_choice_at" json:"finish_final_choice_at"`
	ModifiedAt          time.Time   `bson:"modified_at" json:"modified_at"`
}

type GamerInfo struct {
	UserID             int    `bson:"user_id" json:"user_id"`                             // 사용자 ID
	CharacterID        int    `bson:"character_id" json:"character_id"`                   // 캐릭터 식별자 (0 ~ 5)
	CharacterName      string `bson:"character_avatar_name" json:"character_avatar_name"` // 캐릭터 이름
	CharacterAvatarURL string `bson:"character_avatar_url" json:"character_avatar_url"`   // 캐릭터 아바타 이미지 URL
}
