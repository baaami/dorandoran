package event

import (
	"encoding/json"
	"log"
	"solo/pkg/models"
	"solo/services/match/service"
)

type EventHandler struct {
	service *service.MatchService
}

func NewEventHandler(service *service.MatchService) *EventHandler {
	return &EventHandler{service: service}
}

func (h *EventHandler) HandleRoomCreateEvent(body json.RawMessage) {
	var chatRoom models.ChatRoom
	err := json.Unmarshal(body, &chatRoom)
	if err != nil {
		log.Printf("failed to unmarshal room.create event: %v", err)
		return
	}

	h.service.SendMatchSuccessMessage(chatRoom.UserIDs, chatRoom.ID)
}
