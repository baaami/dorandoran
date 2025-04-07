package printer

import (
	"log"
	"solo/pkg/models"
	eventtypes "solo/pkg/types/eventtype"
)

// ChatEvent 로그 출력
func PrintChatEvent(event eventtypes.ChatEvent) {
	log.Printf("💬 ChatEvent - MessageID: %s, Type: %s, RoomID: %s, SenderID: %d, Message: %s, UnreadCount: %d, CreatedAt: %s, BalanceFormID: %s",
		event.MessageId,
		event.Type,
		event.RoomID,
		event.SenderID,
		event.Message,
		event.UnreadCount,
		event.CreatedAt,
		event.BalanceFormID,
	)
}

// MatchEvent 로그 출력
func PrintMatchEvent(event eventtypes.MatchEvent) {
	log.Printf("🤝 MatchEvent - MatchID: %s",
		event.MatchId,
	)
}

// RoomTimeoutEvent 로그 출력
func PrintRoomTimeoutEvent(event eventtypes.RoomTimeoutEvent) {
	log.Printf("⏰ RoomTimeoutEvent - RoomID: %s",
		event.RoomID,
	)
}

// FinalChoiceTimeoutEvent 로그 출력
func PrintFinalChoiceTimeoutEvent(event eventtypes.FinalChoiceTimeoutEvent) {
	log.Printf("⌛ FinalChoiceTimeoutEvent - RoomID: %s",
		event.RoomID,
	)
}

// RoomJoinEvent 로그 출력
func PrintRoomJoinEvent(event eventtypes.RoomJoinEvent) {
	log.Printf("👋 RoomJoinEvent - RoomID: %s, UserID: %d, JoinAt: %s",
		event.RoomID,
		event.UserID,
		event.JoinAt,
	)
}

// ChatRoom 로그 출력
func PrintChatRoom(room models.ChatRoom) {
	log.Printf("🏠 ChatRoom - ID: %s, Seq: %d, Status: %d, UserIDs: %v, CreatedAt: %s",
		room.ID,
		room.Seq,
		room.Status,
		room.UserIDs,
		room.CreatedAt,
	)
}

// BalanceGameForm 로그 출력
func PrintBalanceGameForm(form models.BalanceGameForm) {
	log.Printf("⚖️ BalanceGameForm - ID: %s, RoomID: %s, Question: %s",
		form.ID.Hex(),
		form.RoomID,
		form.Question,
	)
}

// Error 로그 출력
func PrintError(message string, err error) {
	log.Printf("❌ %s: %v", message, err)
}

// Success 로그 출력
func PrintSuccess(message string) {
	log.Printf("✅ %s", message)
}
