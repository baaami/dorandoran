package event

import (
	"encoding/json"
	"log"
	"solo/pkg/redis"
	"solo/pkg/types/commontype"
	eventtypes "solo/pkg/types/eventtype"
	"solo/services/chat/service"
)

type EventHandler struct {
	chatService *service.ChatService
	redisClient *redis.RedisClient
}

func NewEventHandler(chatService *service.ChatService, redisClient *redis.RedisClient) *EventHandler {
	return &EventHandler{chatService: chatService, redisClient: redisClient}
}

func (e *EventHandler) HandleMatchEvent(body json.RawMessage) {
	var eventData eventtypes.MatchEvent
	if err := json.Unmarshal(body, &eventData); err != nil {
		log.Printf("❌ Failed to unmarshal match event: %v", err)
		return
	}

	log.Printf("[DEBUG] Processing MatchEvent: %+v", eventData)

	err := e.chatService.CreateRoom(eventData)
	if err != nil {
		log.Printf("Failed to create room: %v", err)
	}
}

func (e *EventHandler) HandleRoomTimeout(body json.RawMessage) {
	var eventData eventtypes.RoomTimeoutEvent
	if err := json.Unmarshal(body, &eventData); err != nil {
		log.Printf("❌ Failed to unmarshal room timeout event: %v", err)
		return
	}

	log.Printf("🎯 [DEBUG] Processing RoomTimeoutEvent: %+v", eventData)

	err := e.chatService.UpdateChatRoomStatus(eventData.RoomID, commontype.RoomStatusChoiceIng)
	if err != nil {
		log.Printf("Failed to update chat room status: %d, err: %s", commontype.RoomStatusChoiceIng, err.Error())
	}
}

func (e *EventHandler) HandleFinalChoiceTimeout(body json.RawMessage) {
	var eventData eventtypes.FinalChoiceTimeoutEvent
	if err := json.Unmarshal(body, &eventData); err != nil {
		log.Printf("❌ Failed to unmarshal final choice timeout event: %v", err)
		return
	}

	log.Printf("🎯 [DEBUG] Processing RoomTimeoutEvent: %+v", eventData)

	err := e.chatService.UpdateChatRoomStatus(eventData.RoomID, commontype.RoomStatusChoiceComplete)
	if err != nil {
		log.Printf("Failed to update chat room status: %d, err: %s", commontype.RoomStatusChoiceComplete, err.Error())
	}

	err = e.redisClient.RemoveRoomFromRedis(eventData.RoomID)
	if err != nil {
		log.Printf("Failed to remove RoomID %s from rooms list, err: %v", eventData.RoomID, err)
	}
}

func (e *EventHandler) HandleRoomJoin(body json.RawMessage) {
	var eventData eventtypes.RoomJoinEvent
	if err := json.Unmarshal(body, &eventData); err != nil {
		log.Printf("❌ Failed to unmarshal room join event: %v", err)
		return
	}

	log.Printf("🎯 [DEBUG] Processing RoomJoinEvent: %+v", eventData)

	err := e.chatService.HandleRoomJoin(eventData.RoomID, eventData.UserID, eventData.JoinAt)
	if err != nil {
		log.Printf("Failed to room join, err: %v", err)
	}
}
