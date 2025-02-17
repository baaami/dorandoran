package commontype

import (
	"time"
)

const (
	UserServiceBaseURL = "http://user-service"
)

const (
	ChatTypeChat  = "chat"
	ChatTypeJoin  = "join"
	ChatTypeLeave = "leave"
)

const (
	MALE = iota
	FEMALE
)

const (
	KAKAO = iota
	NAVER
)

const (
	MATCH_COUNT_MIN = 1
	MATCH_COUNT_MAX = 6
)

const (
	MATCH_GAME = iota
	MATCH_COUPLE
)

const (
	USER_STATUS_STANDBY = iota
	USER_STATUS_GAME_ING
)

const DEFAULT_GAME_POINT = 10

const (
	RoomStatusGameIng = iota
	RoomStatusChoiceIng
	RoomStatusChoiceComplete
)

const (
	YoungSoo = iota // 0부터 시작
	YoungHo
	YoungSik
	YoungChul
	KwangSoo
	SangChul
)

var MaleNames = map[int]string{
	YoungSoo:  "영수",
	YoungHo:   "영호",
	YoungSik:  "영식",
	YoungChul: "영철",
	KwangSoo:  "광수",
	SangChul:  "상철",
}

const (
	YoungSook = iota // 0부터 시작
	JungSook
	SoonJa
	YoungJa
	OkSoon
	HyunSook
)

var FemaleNames = map[int]string{
	YoungSook: "영숙",
	JungSook:  "정숙",
	SoonJa:    "순자",
	YoungJa:   "영자",
	OkSoon:    "옥순",
	HyunSook:  "현숙",
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

type User struct {
	ID         int     `gorm:"primaryKey;autoIncrement" json:"id"`
	SnsType    int     `gorm:"index" json:"sns_type"`
	SnsID      string  `gorm:"index" json:"sns_id"`
	Name       string  `gorm:"size:100" json:"name"`
	Gender     int     `json:"gender"`
	Birth      string  `gorm:"size:20" json:"birth"`
	Address    Address `gorm:"embedded;embeddedPrefix:address_" json:"address"`
	GameStatus int     `gorm:"default:0" json:"game_status"`
	GameRoomID string  `gorm:"size:100" json:"game_room_id"`
	GamePoint  int     `json:"game_point"`
}

type MatchFilter struct {
	UserID          int  `gorm:"primaryKey" json:"user_id"`
	CoupleCount     int  `json:"couple_count"`
	AddressRangeUse bool `json:"address_range_use"`
	AgeGroupUse     bool `json:"age_group_use"`
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

type GameInfo struct {
	CharacterID        int    `gorm:"index" json:"character_id"`
	CharacterName      string `gorm:"size:64" json:"character_name"`
	CharacterAvatarURL string `gorm:"size:256" json:"character_avatar_url"`
}

type Gamer struct {
	ID       int      `gorm:"primaryKey;autoIncrement" json:"id"`
	SnsType  int      `gorm:"index" json:"sns_type"`
	SnsID    string   `gorm:"index" json:"sns_id"`
	Name     string   `gorm:"size:100" json:"name"`
	Gender   int      `json:"gender"`
	Birth    string   `gorm:"size:20" json:"birth"`
	Address  Address  `gorm:"embedded;embeddedPrefix:address_" json:"address"`
	GameInfo GameInfo `gorm:"embedded;embeddedPrefix:game_info_" json:"game_info"`
}
