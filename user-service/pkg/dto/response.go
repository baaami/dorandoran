package dto

type UserGameStatusResponse struct {
	Status int    `json:status`
	RoomID string `json:room_id`
}
