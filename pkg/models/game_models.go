package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BalanceGameForm struct {
	ID       primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Question Question             `bson:"question" json:"question"`
	Votes    Votes                `bson:"votes" json:"votes"`
	Comments []BalanceFormComment `bson:"comments" json:"comments"`
}

type Question struct {
	Title string `json:"title"`
	Red   string `json:"red"`
	Blue  string `json:"blue"`
}

type Votes struct {
	RedCount  int `bson:"red_cnt" json:"red_cnt"`
	BlueCount int `bson:"blue_cnt" json:"blue_cnt"`
}

type BalanceFormComment struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FormID    primitive.ObjectID `bson:"balance_form_id" json:"form_id"`
	SenderID  int                `bson:"sender_id" json:"sender_id"`
	Message   string             `bson:"message" json:"message"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// 투표 기록 저장을 위한 구조체
type BalanceFormVote struct {
	FormID    primitive.ObjectID `bson:"form_id" json:"form_id"`
	UserID    int                `bson:"user_id" json:"user_id"`
	Choiced   int                `bson:"choiced" json:"choiced"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}
