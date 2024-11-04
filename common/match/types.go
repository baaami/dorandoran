package common

type Address struct {
	City     string `gorm:"size:100" json:"city"`
	District string `gorm:"size:100" json:"district"`
	Street   string `gorm:"size:100" json:"street"`
}

type MatchFilter struct {
	UserID      int     `gorm:"primaryKey" json:"user_id"`
	CoupleCount int     `json:"couple_count"`
	AddressUse  bool    `json:"address_use"`
	Address     Address `gorm:"embedded;embeddedPrefix:address_" json:"address"`
	AgeGroupUse bool    `json:"age_group_use"`
}
