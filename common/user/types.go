package common

type User struct {
	ID       int    `json:"id"`
	SnsType  int    `json:"sns_type"`
	SnsID    string `json:"sns_id"`
	Name     string `json:"name"`
	Nickname string `json:"nickname"`
	Gender   int    `json:"gender"`
	Age      int    `json:"age"`
	Email    string `json:"email"`
}
