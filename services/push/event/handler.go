package event

import (
	"encoding/json"
	"fmt"
	"log"
	"solo/pkg/types/commontype"
	eventtypes "solo/pkg/types/eventtype"
	"solo/services/push/onesignal"
	"solo/services/user/service"

	"github.com/samber/lo"
)

type EventHandler struct {
	userService *service.UserService
}

func NewEventHandler(userService *service.UserService) *EventHandler {
	return &EventHandler{userService: userService}
}

// HandleChatEvent는 채팅 이벤트를 처리합니다
func (h *EventHandler) HandleChatEvent(body json.RawMessage) {
	var eventData eventtypes.ChatEvent
	if err := json.Unmarshal(body, &eventData); err != nil {
		log.Printf("❌ Failed to unmarshal chat event: %v", err)
		return
	}

	// 알림 설정이 활성화된 사용자만 필터링
	alertEnabledUsers := h.filterAlertEnabledUsers(eventData.InactiveUserIds)

	// 필터링된 사용자가 없으면 early return
	if len(alertEnabledUsers) == 0 {
		log.Printf("ℹ️ No alert-enabled users found for chat event in room %s", eventData.RoomID)
		return
	}

	// 푸시 알림 페이로드 생성
	payload := createPushPayload(
		alertEnabledUsers,
		commontype.PushNotification{
			Header:  "New Message",
			Content: eventData.Message,
			Url:     fmt.Sprintf("randomChat://game-room/%s", eventData.RoomID),
		},
	)

	// 푸시 알림 전송
	if err := onesignal.Push(payload); err != nil {
		log.Printf("❌ Failed to send push notification: %v", err)
		return
	}

	log.Printf("✅ Chat push notification sent to %d users for room %s",
		len(alertEnabledUsers),
		eventData.RoomID,
	)
}

// HandleRoomTimeoutEvent는 방 타임아웃 이벤트를 처리합니다
func (h *EventHandler) HandleRoomTimeoutEvent(body json.RawMessage) {
	var eventData eventtypes.RoomTimeoutEvent
	if err := json.Unmarshal(body, &eventData); err != nil {
		log.Printf("❌ Failed to unmarshal room timeout event: %v", err)
		return
	}

	// 알림 설정이 활성화된 사용자만 필터링
	alertEnabledUsers := h.filterAlertEnabledUsers(eventData.InactiveUserIds)

	// 필터링된 사용자가 없으면 early return
	if len(alertEnabledUsers) == 0 {
		log.Printf("ℹ️ No alert-enabled users found for timeout event in room %s", eventData.RoomID)
		return
	}

	// 푸시 알림 페이로드 생성
	payload := createPushPayload(
		alertEnabledUsers,
		commontype.PushNotification{
			Header:  "Final Choice Start",
			Content: "Time to make your final choice!",
			Url:     fmt.Sprintf("randomChat://game-room/%s", eventData.RoomID),
		},
	)

	// 푸시 알림 전송
	if err := onesignal.Push(payload); err != nil {
		log.Printf("❌ Failed to send push notification: %v", err)
		return
	}

	log.Printf("✅ Timeout push notification sent to %d users for room %s",
		len(alertEnabledUsers),
		eventData.RoomID,
	)
}

// filterAlertEnabledUsers는 알림 설정이 활성화된 사용자만 필터링
func (h *EventHandler) filterAlertEnabledUsers(userIDs []int) []int {
	return lo.Filter(userIDs, func(userID int, _ int) bool {
		alert, err := h.userService.GetUserAlert(userID)
		if err != nil {
			log.Printf("⚠️ Failed to get user alert setting for user %d: %v", userID, err)
			return false
		}
		return alert
	})
}

// createPushPayload는 푸시 알림 페이로드를 생성
func createPushPayload(users []int, notification commontype.PushNotification) onesignal.Payload {
	return onesignal.Payload{
		PushUserList: users,
		Header:       notification.Header,
		Content:      notification.Content,
		Url:          notification.Url,
	}
}
