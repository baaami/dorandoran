package event

import (
	"encoding/json"
	"solo/pkg/models"
	"solo/pkg/redis"
	"solo/pkg/types/commontype"
	eventtypes "solo/pkg/types/eventtype"
	"solo/pkg/utils/printer"
	"solo/services/chat/service"
	"time"

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
		printer.PrintError("Failed to unmarshal chat event", err)
		return
	}

	printer.PrintChatEvent(chatEvent)

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
		printer.PrintError("Failed to insert chat message", err)
	}
}

func (e *EventHandler) HandleMatchEvent(body json.RawMessage) {
	var eventData eventtypes.MatchEvent
	if err := json.Unmarshal(body, &eventData); err != nil {
		printer.PrintError("Failed to unmarshal match event", err)
		return
	}

	printer.PrintMatchEvent(eventData)

	err := e.chatService.CreateRoom(eventData)
	if err != nil {
		printer.PrintError("Failed to create room", err)
	}
}

func (e *EventHandler) HandleRoomCreateEvent(body json.RawMessage) {
	var chatRoom models.ChatRoom
	err := json.Unmarshal(body, &chatRoom)
	if err != nil {
		printer.PrintError("Failed to unmarshal room.create event", err)
		return
	}

	matchHistory := models.MatchHistory{
		ID:             primitive.NewObjectID(),
		RoomSeq:        int(chatRoom.Seq),
		UserIDs:        chatRoom.UserIDs,
		BalanceResults: []models.BalanceGameResult{},
		FinalMatch:     []string{},
		CreatedAt:      chatRoom.CreatedAt,
	}

	e.chatService.SaveMatchHistory(matchHistory)
}

func (e *EventHandler) HandleRoomTimeout(body json.RawMessage) {
	var eventData eventtypes.RoomTimeoutEvent
	if err := json.Unmarshal(body, &eventData); err != nil {
		printer.PrintError("Failed to unmarshal room timeout event", err)
		return
	}

	printer.PrintRoomTimeoutEvent(eventData)

	err := e.chatService.UpdateChatRoomStatus(eventData.RoomID, commontype.RoomStatusChoiceIng)
	if err != nil {
		printer.PrintError("Failed to update chat room status", err)
	}
}

func (e *EventHandler) cleanupRoomData(roomID string) {
	room, err := e.chatService.GetChatRoomByID(roomID)
	if err != nil {
		printer.PrintError("Failed to get room status", err)
		return
	}

	if room.Status != commontype.RoomStatusGameEnd {
		printer.PrintError("Room is not in GameEnd status", nil)
		return
	}

	balanceForms, err := e.chatService.GetBalanceFormsByRoomID(roomID)
	if err != nil {
		printer.PrintError("Failed to get balance forms", err)
		return
	}

	for _, form := range balanceForms {
		err = e.chatService.DeleteBalanceFormVotes(form.ID)
		if err != nil {
			printer.PrintError("Failed to delete balance form votes", err)
		}

		err = e.chatService.DeleteBalanceFormComments(form.ID)
		if err != nil {
			printer.PrintError("Failed to delete balance form comments", err)
		}
	}

	err = e.chatService.DeleteBalanceFormsByRoomID(roomID)
	if err != nil {
		printer.PrintError("Failed to delete balance forms", err)
	}

	err = e.chatService.DeleteMessageReaders(roomID)
	if err != nil {
		printer.PrintError("Failed to delete message readers", err)
	}

	err = e.chatService.DeleteChatByRoomID(roomID)
	if err != nil {
		printer.PrintError("Failed to delete messages", err)
	}

	err = e.chatService.DeleteChatRoom(roomID)
	if err != nil {
		printer.PrintError("Failed to delete room", err)
	}

	printer.PrintSuccess("Successfully cleaned up all data for room " + roomID)
}

func (e *EventHandler) HandleFinalChoiceTimeout(body json.RawMessage) {
	var eventData eventtypes.FinalChoiceTimeoutEvent
	if err := json.Unmarshal(body, &eventData); err != nil {
		printer.PrintError("Failed to unmarshal final choice timeout event", err)
		return
	}

	printer.PrintFinalChoiceTimeoutEvent(eventData)

	err := e.chatService.UpdateChatRoomStatus(eventData.RoomID, commontype.RoomStatusChoiceComplete)
	if err != nil {
		printer.PrintError("Failed to update chat room status", err)
	}

	err = e.redisClient.RemoveChoiceRoomFromRedis(eventData.RoomID)
	if err != nil {
		printer.PrintError("Failed to remove RoomID from rooms list", err)
	}

	err = e.chatService.UpdateChatRoomStatus(eventData.RoomID, commontype.RoomStatusGameEnd)
	if err != nil {
		printer.PrintError("Failed to update chat room status", err)
	}

	go func() {
		time.Sleep(commontype.RemoveRoomDataTimer)
		e.cleanupRoomData(eventData.RoomID)
	}()
}

func (e *EventHandler) HandleRoomJoin(body json.RawMessage) {
	var eventData eventtypes.RoomJoinEvent
	if err := json.Unmarshal(body, &eventData); err != nil {
		printer.PrintError("Failed to unmarshal room join event", err)
		return
	}

	printer.PrintRoomJoinEvent(eventData)

	err := e.chatService.HandleRoomJoin(eventData.RoomID, eventData.UserID, eventData.JoinAt)
	if err != nil {
		printer.PrintError("Failed to room join", err)
	}
}
