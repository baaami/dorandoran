package dto

type MatchFilterDTO struct {
	UserID          int  `gorm:"primaryKey" json:"user_id"`
	CoupleCount     int  `json:"couple_count"`
	AddressRangeUse bool `json:"address_range_use"`
	AgeGroupUse     bool `json:"age_group_use"`
}
