package commontype

const (
	UserServiceBaseURL = "http://user-service"
)

const (
	KAKAO = iota
	NAVER
)

const (
	USER_STATUS_STANDBY = iota
	USER_STATUS_GAME_ING
)

const DEFAULT_GAME_POINT = 10

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
