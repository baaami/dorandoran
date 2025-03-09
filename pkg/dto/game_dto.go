package dto

import "solo/pkg/models"

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
