package commontype

import "time"

const (
	UserServiceBaseURL = "http://doran-user"
)

const (
	ChatTypeChat  = "chat"
	ChatTypeForm  = "form"
	ChatTypeJoin  = "join"
	ChatTypeLeave = "leave"
)

const (
	MasterID = 0
)

const (
	GameRunningTime        = 1 * time.Hour
	CoupleRunningTime      = 24 * 3 * time.Hour
	FinishFinalChoiceTimer = 30 * time.Second
	BalanceGameTimer       = 15 * time.Minute
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
const DEFAULT_PAGE_SIZE = 20
const DEFAULT_TEMP_SERVER_ID = "game-server-1"

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

type PushNotification struct {
	Header  string
	Content string
	Url     string
}
