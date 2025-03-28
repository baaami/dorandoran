package event

import (
	"encoding/json"
	"log"
	"solo/pkg/models"
	"solo/pkg/redis"
	"solo/pkg/types/commontype"
	eventtypes "solo/pkg/types/eventtype"
	"solo/services/chat/service"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EventHandler struct {
	chatService *service.ChatService
	redisClient *redis.RedisClient
}

func NewEventHandler(chatService *service.ChatService, redisClient *redis.RedisClient) *EventHandler {
	return &EventHandler{chatService: chatService, redisClient: redisClient}
}

func (e *EventHandler) HandleChatEvent(body json.RawMessage) {
	var chatEvent eventtypes.ChatEvent
	if err := json.Unmarshal(body, &chatEvent); err != nil {
		log.Printf("❌ Failed to unmarshal chat event: %v", err)
		return
	}

	log.Printf("💬 [DEBUG] Processing ChatEvent: %+v", chatEvent)

	chat := models.Chat{
		MessageId:     chatEvent.MessageId,
		Type:          chatEvent.Type,
		RoomID:        chatEvent.RoomID,
		SenderID:      chatEvent.SenderID,
		Message:       chatEvent.Message,
		UnreadCount:   chatEvent.UnreadCount,
		CreatedAt:     chatEvent.CreatedAt,
		BalanceFormID: chatEvent.BalanceFormID,
	}

	_, err := e.chatService.AddChatMsg(chat)
	if err != nil {
		log.Printf("❌ Failed to insert chat message, %s", chatEvent.Message)
	}
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

func (e *EventHandler) HandleRoomCreateEvent(body json.RawMessage) {
	var chatRoom models.ChatRoom
	err := json.Unmarshal(body, &chatRoom)
	if err != nil {
		log.Printf("failed to unmarshal room.create event: %v", err)
		return
	}

	matchHistory := models.MatchHistory{
		ID:             primitive.NewObjectID(),
		RoomSeq:        int(chatRoom.Seq),
		UserIDs:        chatRoom.UserIDs,
		BalanceResults: []models.BalanceGameResult{}, // 밸런스 게임 종료 시 업데이트
		FinalMatch:     []string{},                   // 최종 선택 이후 업데이트
		CreatedAt:      chatRoom.CreatedAt,
	}

	e.chatService.SaveMatchHistory(matchHistory)
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

	log.Printf("🎯 [DEBUG] Processing FinalChoiceTimeoutEvent: %+v", eventData)

	err := e.chatService.UpdateChatRoomStatus(eventData.RoomID, commontype.RoomStatusChoiceComplete)
	if err != nil {
		log.Printf("Failed to update chat room status: %d, err: %s", commontype.RoomStatusChoiceComplete, err.Error())
	}

	err = e.redisClient.RemoveChoiceRoomFromRedis(eventData.RoomID)
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
