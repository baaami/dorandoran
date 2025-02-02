package dto

type AddressDTO struct {
	City     string `json:"city"`
	District string `json:"district"`
	Street   string `json:"street"`
}

type UserDTO struct {
	ID         int        `json:"id"`
	SnsType    int        `json:"sns_type"`
	SnsID      string     `json:"sns_id"`
	Name       string     `json:"name"`
	Gender     int        `json:"gender"`
	Birth      string     `json:"birth"`
	Address    AddressDTO `json:"address"`
	GameStatus int        `json:"game_status"`
	GameRoomID string     `json:"game_room_id"`
	GamePoint  int        `json:"game_point"`
}
