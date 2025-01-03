package types

const (
	MATCH_GAME = iota
	MATCH_COUPLE
)

const (
	MALE = iota
	FEMALE
)

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

type MatchEvent struct {
	MatchId      string        `bson:"match_id" json:"match_id"`
	MatchType    int           `bson:"match_type" json:"match_type"`
	MatchedUsers []WaitingUser `bson:"matched_users" json:"matched_users"`
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

type Gamer struct {
	ID            int     `gorm:"primaryKey;autoIncrement" json:"id"`
	SnsType       int     `gorm:"index" json:"sns_type"`
	SnsID         string  `gorm:"index" json:"sns_id"`
	Name          string  `gorm:"size:100" json:"name"`
	Gender        int     `json:"gender"`
	Birth         string  `gorm:"size:20" json:"birth"`
	Address       Address `gorm:"embedded;embeddedPrefix:address_" json:"address"`
	GameID        int     `gorm:"index" json:"game_id"`
	GameAvatarURL string  `gorm:"size:256" json:"game_avatar_url"`
}
