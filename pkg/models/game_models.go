package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BalanceGameForm struct {
	ID       int       `json:"id"`
	Question Question  `json:"question"`
	Votes    Votes     `json:"votes"`
	Comments []Comment `json:"comments"`
}

type Question struct {
	Title string `json:"title"`
	Red   string `json:"red"`
	Blue  string `json:"blue"`
}

type Votes struct {
	RedCount  int `json:"red_cnt"`
	BlueCount int `json:"blue_cnt"`
}

type Comment struct {
	SenderID  int       `json:"sender_id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// 투표 기록 저장을 위한 구조체
type BalanceFormVote struct {
	FormID    primitive.ObjectID `bson:"form_id"`
	UserID    int                `bson:"user_id"`
	IsRed     bool               `bson:"is_red"`
	CreatedAt time.Time          `bson:"created_at"`
}
