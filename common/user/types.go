package common

type User struct {
	ID       int    `gorm:"primaryKey;autoIncrement" json:"id"`
	SnsType  int    `gorm:"index" json:"sns_type"`
	SnsID    int64  `gorm:"index" json:"sns_id"`
	Name     string `gorm:"size:100" json:"name"`
	Nickname string `gorm:"size:100" json:"nickname"`
	Gender   int    `json:"gender"`
	Age      int    `json:"age"`
	Email    string `gorm:"size:100" json:"email"`
}
