package types

type Address struct {
	City     string `gorm:"size:100" json:"city"`
	District string `gorm:"size:100" json:"district"`
	Street   string `gorm:"size:100" json:"street"`
}

type WaitingUser struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	Gender          int     `json:"gender"`
	Birth           string  `json:"birth"`
	Address         Address `json:"address"`
	CoupleCount     int     `json:"couple_count"`
	AddressRangeUse bool    `json:"address_range_use"`
	AgeGroupUse     bool    `json:"age_group_use"`
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