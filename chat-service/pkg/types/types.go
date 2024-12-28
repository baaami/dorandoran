package types

const (
	MATCH_GAME = iota
	MATCH_COUPLE
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
