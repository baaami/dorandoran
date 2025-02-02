package event

import (
	"encoding/json"
	"log"
	"solo/pkg/types/commontype"
	eventtypes "solo/pkg/types/eventtype"
	"solo/services/user/service"
)

type EventHandler struct {
	userService *service.UserService
}

func NewEventHandler(userService *service.UserService) *EventHandler {
	return &EventHandler{userService: userService}
}

func (h *EventHandler) HandleMatchEvent(body []byte) {
	var eventData eventtypes.MatchEvent
	if err := json.Unmarshal(body, &eventData); err != nil {
		log.Printf("❌ Failed to unmarshal match event: %v", err)
		return
	}

	log.Printf("🎯 Processing Match Event: %+v", eventData)

	for _, user := range eventData.MatchedUsers {
		err := h.userService.UpdateUserGameInfo(user.ID, commontype.USER_STATUS_GAME_ING, eventData.MatchId)
		if err != nil {
			log.Printf("❌ Failed to update user status: %v", err)
			continue
		}
	}
}

func (h *EventHandler) HandleFinalChoiceTimeout(body []byte) {
	var eventData eventtypes.FinalChoiceTimeoutEvent
	if err := json.Unmarshal(body, &eventData); err != nil {
		log.Printf("❌ Failed to unmarshal final choice timeout event: %v", err)
		return
	}

	log.Printf("⏳ Handling Final Choice Timeout for users: %+v", eventData.UserIDs)

	for _, id := range eventData.UserIDs {
		err := h.userService.UpdateUserGameInfo(id, commontype.USER_STATUS_STANDBY, "")
		if err != nil {
			log.Printf("❌ Failed to update user status for ID %d: %v", id, err)
		}
	}
}

func (h *EventHandler) HandleRoomLeave(body []byte) {
	var eventData eventtypes.RoomLeaveEvent
	if err := json.Unmarshal(body, &eventData); err != nil {
		log.Printf("❌ Failed to unmarshal room leave event: %v", err)
		return
	}

	log.Printf("🏠 User %d left the room, updating status...", eventData.LeaveUserID)

	err := h.userService.UpdateUserGameInfo(eventData.LeaveUserID, commontype.USER_STATUS_STANDBY, "")
	if err != nil {
		log.Printf("❌ Failed to update user status: %v", err)
	}
}
