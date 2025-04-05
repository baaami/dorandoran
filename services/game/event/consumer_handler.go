package event

import (
	"encoding/json"
	"log"
	"solo/pkg/helper"
	"solo/pkg/models"
	"solo/pkg/redis"
	eventtypes "solo/pkg/types/eventtype"
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
		log.Printf("❌ Failed to unmarshal chat event: %v", err)
		return
	}

	log.Printf("💬 [DEBUG] Processing ChatEvent: %+v", chatEvent)

	wsMessage := stype.WebSocketMessage{
		Kind:    stype.MessageKindMessage,
		Payload: payload,
	}

	// 메시지를 WebSocket으로 전송
	err := e.gameService.SendMessageToRoom(chatEvent.RoomID, wsMessage)
	if err != nil {
		log.Printf("❌ Failed to send message via WebSocket: %v", err)
	}
}

func (e *EventHandler) HandleVoteChatEvent(payload json.RawMessage) {
	var chatEvent eventtypes.ChatEvent
	if err := json.Unmarshal(payload, &chatEvent); err != nil {
		log.Printf("❌ Failed to unmarshal chat event: %v", err)
		return
	}

	log.Printf("💬 [DEBUG] Processing ChatEvent: %+v", chatEvent)

	wsMessage := stype.WebSocketMessage{
		Kind:    stype.MessageKindVoteCommentMessage,
		Payload: payload,
	}

	// 메시지를 WebSocket으로 전송
	err := e.gameService.SendMessageToRoom(chatEvent.RoomID, wsMessage)
	if err != nil {
		log.Printf("❌ Failed to send message via WebSocket: %v", err)
	}
}

func (e *EventHandler) HandleChatLatestEvent(payload json.RawMessage) {
	var chatLatestEvent eventtypes.ChatLatestEvent
	if err := json.Unmarshal(payload, &chatLatestEvent); err != nil {
		log.Printf("❌ Failed to unmarshal chat.latest event: %v", err)
		return
	}

	log.Printf("📢 Broadcasting chat.latest for Room %s", chatLatestEvent.RoomID)

	wsMessage := stype.WebSocketMessage{
		Kind:    stype.MessageKindChatLastest,
		Payload: payload,
	}

	err := e.gameService.SendMessageToRoom(chatLatestEvent.RoomID, wsMessage)
	if err != nil {
		log.Printf("❌ Failed to send chat.latest message via WebSocket: %v", err)
	}
}

func (e *EventHandler) HandleCoupleRoomCreateEvent(payload json.RawMessage) {
	var chatRoom models.ChatRoom
	if err := json.Unmarshal(payload, &chatRoom); err != nil {
		log.Printf("❌ Failed to unmarshal chat room create event: %v", err)
		return
	}

	coupleRoom := stype.CoupleMatchSuccessMessage{
		RoomID: chatRoom.ID,
	}

	log.Printf("💖 Couple Room Created, Room ID: %s", coupleRoom.RoomID)

	wsMessage := stype.WebSocketMessage{
		Kind:    stype.MessageKindCoupleMatchSuccess,
		Payload: helper.ToJSON(coupleRoom),
	}

	err := e.gameService.SendMessageToRoom(coupleRoom.RoomID, wsMessage)
	if err != nil {
		log.Printf("❌ Failed to send Couple Match Success message via WebSocket: %v", err)
	}
}

func (e *EventHandler) HandleRoomLeaveEvent(payload json.RawMessage) {
	var roomLeave eventtypes.RoomLeaveEvent
	if err := json.Unmarshal(payload, &roomLeave); err != nil {
		log.Printf("❌ Failed to unmarshal room leave event: %v", err)
		return
	}

	log.Printf("📢 Broadcasting room leave event, Room ID: %s, User ID: %v", roomLeave.RoomID, roomLeave.LeaveUserID)

	wsMessage := stype.WebSocketMessage{
		Kind:    stype.MessageKindLeave,
		Payload: payload,
	}

	err := e.gameService.SendMessageToRoom(roomLeave.RoomID, wsMessage)
	if err != nil {
		log.Printf("❌ Failed to send room leave message via WebSocket: %v", err)
	}
}

func (e *EventHandler) HandleRoomTimeoutEvent(payload json.RawMessage) {
	var roomTimeout eventtypes.RoomTimeoutEvent
	if err := json.Unmarshal(payload, &roomTimeout); err != nil {
		log.Printf("❌ Failed to unmarshal room timeout event: %v", err)
		return
	}

	log.Printf("⏳ Processing Room Timeout for Room %s", roomTimeout.RoomID)

	err := e.gameService.BroadCastFinalChoiceStart(roomTimeout.RoomID)
	if err != nil {
		log.Printf("❌ Failed to process Room Timeout: %v", err)
	}
}

func (e *EventHandler) HandleFinalChoiceTimeoutEvent(payload json.RawMessage) {
	var finalChoiceTimeout eventtypes.FinalChoiceEvent
	if err := json.Unmarshal(payload, &finalChoiceTimeout); err != nil {
		log.Printf("❌ Failed to unmarshal final choice timeout event: %v", err)
		return
	}

	log.Printf("💡 Broadcasting final choice timeout for Room %s", finalChoiceTimeout.RoomID)

	err := e.gameService.BroadcastFinalChoices(finalChoiceTimeout.RoomID)
	if err != nil {
		log.Printf("❌ Failed to broadcast final choice room: %v", err)
	}
}

func (e *EventHandler) HandleVoteCommentChatEvent(payload json.RawMessage) {
	var voteCommentChatEvent eventtypes.VoteCommentChatEvent
	if err := json.Unmarshal(payload, &voteCommentChatEvent); err != nil {
		log.Printf("❌ Failed to unmarshal vote comment chat event: %v", err)
		return
	}

	log.Printf("💬 [DEBUG] Processing VoteCommentChatEvent: %+v", voteCommentChatEvent)

	wsMessage := stype.WebSocketMessage{
		Kind:    stype.MessageKindVoteCommentMessage,
		Payload: payload,
	}

	err := e.gameService.SendMessageToRoom(voteCommentChatEvent.RoomID, wsMessage)
	if err != nil {
		log.Printf("❌ Failed to send vote comment chat message via WebSocket: %v", err)
	}
}
