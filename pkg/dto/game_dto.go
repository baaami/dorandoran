package dto

import (
	"solo/pkg/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BalanceFormVoteDTO struct {
	Choiced int `json:"choiced"`
}

type BalanceFormCommentDTO struct {
	Comment string `json:"comment"`
}

type BalanceFormCommentListResponse struct {
	Data        []models.BalanceFormComment `json:"data"`
	CurrentPage int                         `json:"current_page"`
	NextPage    int                         `json:"next_page"`
	HasNextPage bool                        `json:"has_next_page"`
	TotalPages  int                         `json:"total_pages"`
}

type BalanceGameFormDTO struct {
	ID       primitive.ObjectID          `bson:"_id,omitempty" json:"id"`
	RoomID   string                      `bson:"room_id" json:"room_id"`
	Question models.Question             `bson:"question" json:"question"`
	Votes    models.Votes                `bson:"votes" json:"votes"`
	Comments []models.BalanceFormComment `bson:"comments" json:"comments"`
	MyVote   int                         `bson:"my_vote" json:"my_vote"` // -1: 투표 안함, 0: 레드, 1: 블루
}
