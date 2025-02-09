package event

import (
	"encoding/json"
	"fmt"
	"log"
	eventtypes "solo/pkg/types/eventtype"
	"solo/services/push/onesignal"
)

type EventHandler struct {
}

func NewEventHandler() *EventHandler {
	return &EventHandler{}
}

func (h *EventHandler) HandleChatEvent(body json.RawMessage) {
	var eventData eventtypes.ChatEvent
	if err := json.Unmarshal(body, &eventData); err != nil {
		log.Printf("❌ Failed to unmarshal chat event: %v", err)
		return
	}

	payload := onesignal.Payload{
		PushUserList: eventData.InactiveUserIds,
		Header:       "New Message",
		Content:      eventData.Message,
		Url:          fmt.Sprintf("randomChat://game-room/%s", eventData.RoomID),
	}

	onesignal.Push(payload)

	log.Printf("🎯 Processing Chat Event: %+v", eventData)
}

func (h *EventHandler) HandleRoomTimeoutEvent(body json.RawMessage) {
	var eventData eventtypes.RoomTimeoutEvent
	if err := json.Unmarshal(body, &eventData); err != nil {
		log.Printf("❌ Failed to unmarshal room timeout event: %v", err)
		return
	}

	payload := onesignal.Payload{
		PushUserList: eventData.InactiveUserIds,
		Header:       "Final Choice Start",
		Content:      "Final Choice Start",
		Url:          fmt.Sprintf("randomChat://game-room/%s", eventData.RoomID),
	}

	onesignal.Push(payload)

	log.Printf("⏳ Handling Room Timeout, room: %s", eventData.RoomID)
}
