package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BalanceGameForm struct {
	ID       primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	RoomID   string               `bson:"room_id" json:"room_id"`
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

type BalanceFormVote struct {
	FormID    primitive.ObjectID `bson:"form_id" json:"form_id"`
	UserID    int                `bson:"user_id" json:"user_id"`
	Choiced   int                `bson:"choiced" json:"choiced"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

type BalanceGameResult struct {
	GameID     primitive.ObjectID `bson:"balance_game_id" json:"balance_game_id"` // 밸런스 게임 ID
	WinnerTeam int                `bson:"winner_team" json:"winner_team"`         // 승리 팀 (0: red, 1: blue)
}

type MatchHistory struct {
	ID             primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	RoomSeq        int                 `bson:"room_seq" json:"room_seq"`
	UserIDs        []int               `bson:"user_ids" json:"user_ids"`
	BalanceResults []BalanceGameResult `bson:"balance_results" json:"balance_results"` // 최종 선택 완료 후 업데이트
	FinalMatch     []string            `bson:"final_match" json:"final_match"`         // 최종 선택 완료 후 업데이트
	CreatedAt      time.Time           `bson:"created_at" json:"created_at"`
}

type BalanceGame struct {
	ID    primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title string             `bson:"title" json:"title"`
	Red   string             `bson:"red" json:"red"`
	Blue  string             `bson:"blue" json:"blue"`
}
