package event

import (
	"encoding/json"
	"fmt"
	"solo/pkg/helper"
	"solo/pkg/models"
	"solo/pkg/redis"
	eventtypes "solo/pkg/types/eventtype"
	"solo/pkg/utils/printer"
	"solo/pkg/utils/stype"

	"solo/services/game/service"
)

type EventHandler struct {
	gameService *service.GameService
	redisClient *redis.RedisClient
}

func NewEventHandler(gameService *service.GameService, redisClient *redis.RedisClient) *EventHandler {
	return &EventHandler{gameService: gameService, redisClient: redisClient}
}

func (e *EventHandler) HandleChatEvent(payload json.RawMessage) {
	var chatEvent eventtypes.ChatEvent
	if err := json.Unmarshal(payload, &chatEvent); err != nil {
		printer.PrintError("Failed to unmarshal chat event", err)
		return
	}

	printer.PrintChatEvent(chatEvent)

	wsMessage := stype.WebSocketMessage{
		Kind:    stype.MessageKindMessage,
		Payload: payload,
	}

	err := e.gameService.SendMessageToRoom(chatEvent.RoomID, wsMessage)
	if err != nil {
		printer.PrintError("Failed to send message via WebSocket", err)
	}
}

func (e *EventHandler) HandleVoteChatEvent(payload json.RawMessage) {
	var chatEvent eventtypes.ChatEvent
	if err := json.Unmarshal(payload, &chatEvent); err != nil {
		printer.PrintError("Failed to unmarshal chat event", err)
		return
	}

	printer.PrintChatEvent(chatEvent)

	wsMessage := stype.WebSocketMessage{
		Kind:    stype.MessageKindVoteCommentMessage,
		Payload: payload,
	}

	err := e.gameService.SendMessageToRoom(chatEvent.RoomID, wsMessage)
	if err != nil {
		printer.PrintError("Failed to send message via WebSocket", err)
	}
}

func (e *EventHandler) HandleChatLatestEvent(payload json.RawMessage) {
	var chatLatestEvent eventtypes.ChatLatestEvent
	if err := json.Unmarshal(payload, &chatLatestEvent); err != nil {
		printer.PrintError("Failed to unmarshal chat.latest event", err)
		return
	}

	printer.PrintSuccess("Broadcasting chat.latest for Room " + chatLatestEvent.RoomID)

	wsMessage := stype.WebSocketMessage{
		Kind:    stype.MessageKindChatLastest,
		Payload: payload,
	}

	err := e.gameService.SendMessageToRoom(chatLatestEvent.RoomID, wsMessage)
	if err != nil {
		printer.PrintError("Failed to send chat.latest message via WebSocket", err)
	}
}

func (e *EventHandler) HandleCoupleRoomCreateEvent(payload json.RawMessage) {
	var chatRoom models.ChatRoom
	if err := json.Unmarshal(payload, &chatRoom); err != nil {
		printer.PrintError("Failed to unmarshal chat room create event", err)
		return
	}

	coupleRoom := stype.CoupleMatchSuccessMessage{
		RoomID: chatRoom.ID,
	}

	printer.PrintSuccess("Couple Room Created, Room ID: " + coupleRoom.RoomID)

	wsMessage := stype.WebSocketMessage{
		Kind:    stype.MessageKindCoupleMatchSuccess,
		Payload: helper.ToJSON(coupleRoom),
	}

	err := e.gameService.SendMessageToRoom(coupleRoom.RoomID, wsMessage)
	if err != nil {
		printer.PrintError("Failed to send Couple Match Success message via WebSocket", err)
	}
}

func (e *EventHandler) HandleRoomLeaveEvent(payload json.RawMessage) {
	var roomLeave eventtypes.RoomLeaveEvent
	if err := json.Unmarshal(payload, &roomLeave); err != nil {
		printer.PrintError("Failed to unmarshal room leave event", err)
		return
	}

	printer.PrintSuccess(fmt.Sprintf("Broadcasting room leave event, Room ID: %s, User ID: %d", roomLeave.RoomID, roomLeave.LeaveUserID))

	wsMessage := stype.WebSocketMessage{
		Kind:    stype.MessageKindLeave,
		Payload: payload,
	}

	err := e.gameService.SendMessageToRoom(roomLeave.RoomID, wsMessage)
	if err != nil {
		printer.PrintError("Failed to send room leave message via WebSocket", err)
	}
}

func (e *EventHandler) HandleRoomTimeoutEvent(payload json.RawMessage) {
	var roomTimeout eventtypes.RoomTimeoutEvent
	if err := json.Unmarshal(payload, &roomTimeout); err != nil {
		printer.PrintError("Failed to unmarshal room timeout event", err)
		return
	}

	printer.PrintSuccess("Processing Room Timeout for Room " + roomTimeout.RoomID)

	err := e.gameService.BroadCastFinalChoiceStart(roomTimeout.RoomID)
	if err != nil {
		printer.PrintError("Failed to process Room Timeout", err)
	}
}

func (e *EventHandler) HandleFinalChoiceTimeoutEvent(payload json.RawMessage) {
	var finalChoiceTimeout eventtypes.FinalChoiceEvent
	if err := json.Unmarshal(payload, &finalChoiceTimeout); err != nil {
		printer.PrintError("Failed to unmarshal final choice timeout event", err)
		return
	}

	printer.PrintSuccess("Broadcasting final choice timeout for Room " + finalChoiceTimeout.RoomID)

	err := e.gameService.BroadcastFinalChoices(finalChoiceTimeout.RoomID)
	if err != nil {
		printer.PrintError("Failed to broadcast final choice room", err)
	}
}

func (e *EventHandler) HandleVoteCommentChatEvent(payload json.RawMessage) {
	var voteCommentChatEvent eventtypes.VoteCommentChatEvent
	if err := json.Unmarshal(payload, &voteCommentChatEvent); err != nil {
		printer.PrintError("Failed to unmarshal vote comment chat event", err)
		return
	}

	printer.PrintSuccess(fmt.Sprintf("Processing vote comment chat event - FormID: %s, RoomID: %s",
		voteCommentChatEvent.FormID.Hex(),
		voteCommentChatEvent.RoomID))

	wsMessage := stype.WebSocketMessage{
		Kind:    stype.MessageKindVoteCommentMessage,
		Payload: payload,
	}

	err := e.gameService.SendMessageToRoom(voteCommentChatEvent.RoomID, wsMessage)
	if err != nil {
		printer.PrintError("Failed to send vote comment chat message via WebSocket", err)
	}
}
