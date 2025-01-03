package types

import (
	"encoding/json"
	"time"
)

type EventPayload struct {
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

type Address struct {
	City     string `gorm:"size:100" json:"city"`
	District string `gorm:"size:100" json:"district"`
	Street   string `gorm:"size:100" json:"street"`
}

type WaitingUser struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Gender      int     `json:"gender"`
	Birth       string  `json:"birth"`
	Address     Address `json:"address"`
	CoupleCount int     `json:"couple_count"`
}

type MatchFilter struct {
	UserID          int  `gorm:"primaryKey" json:"user_id"`
	CoupleCount     int  `json:"couple_count"`
	AddressRangeUse bool `json:"address_range_use"`
	AgeGroupUse     bool `json:"age_group_use"`
}

type User struct {
	ID      int     `gorm:"primaryKey;autoIncrement" json:"id"`
	SnsType int     `gorm:"index" json:"sns_type"`
	SnsID   string  `gorm:"index" json:"sns_id"`
	Name    string  `gorm:"size:100" json:"name"`
	Gender  int     `json:"gender"`
	Birth   string  `gorm:"size:20" json:"birth"`
	Address Address `gorm:"embedded;embeddedPrefix:address_" json:"address"`
}

type MatchEvent struct {
	MatchId      string        `bson:"match_id" json:"match_id"`
	MatchType    int           `bson:"match_type" json:"match_type"`
	MatchedUsers []WaitingUser `bson:"matched_users" json:"matched_users"`
}

type ChatRoom struct {
	ID           string      `bson:"id" json:"id"` // UUID 사용
	Type         int         `bson:"type" json:"type"`
	UserIDs      []int       `bson:"user_ids" json:"user_ids"`
	Gamers       []GamerInfo `bson:"gamers" json:"gamers"` // 사용자별 캐릭터 정보
	CreatedAt    time.Time   `bson:"created_at" json:"created_at"`
	FinishChatAt time.Time   `bson:"finish_chat_at" json:"finish_chat_at"`
	ModifiedAt   time.Time   `bson:"modified_at" json:"modified_at"`
}

// 사용자별 캐릭터 정보를 저장하는 구조체
type GamerInfo struct {
	UserID      int    `bson:"user_id" json:"user_id"`           // 사용자 ID
	CharacterID int    `bson:"character_id" json:"character_id"` // 캐릭터 식별자 (0 ~ 5)
	AvatarURL   string `bson:"avatar_url" json:"avatar_url"`     // 캐릭터 아바타 이미지 URL
}
