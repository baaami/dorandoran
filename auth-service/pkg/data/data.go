package data

type Address struct {
	City     string `gorm:"size:100" json:"city"`
	District string `gorm:"size:100" json:"district"`
	Street   string `gorm:"size:100" json:"street"`
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
