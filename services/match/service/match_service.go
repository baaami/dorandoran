package service

import (
	"encoding/json"
	"fmt"
	"log"
	"solo/pkg/dto"
	"solo/pkg/helper"
	"solo/pkg/logger"
	"solo/pkg/mq"
	"solo/pkg/redis"
	"solo/pkg/utils/stype"

	"solo/pkg/types/commontype"
	eventtypes "solo/pkg/types/eventtype"

	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/samber/lo"
)

type MQEmitter interface {
	PublishMatchEvent(eventtypes.EventPayload) error
}

type MatchService struct {
	redisClient  *redis.RedisClient
	mqClient     *mq.RabbitMQ
	MatchClients sync.Map
	emitter      MQEmitter
}

func NewMatchService(redisClient *redis.RedisClient, mqClient *mq.RabbitMQ, emitter MQEmitter) *MatchService {
	service := &MatchService{
		redisClient: redisClient,
		mqClient:    mqClient,
		emitter:     emitter,
	}

	go service.startMatchMonitoring()

	return service
}

func (s *MatchService) RegisterUserToMatch(conn *websocket.Conn, waitingUser commontype.WaitingUser) error {
	_, ok := s.MatchClients.Load(waitingUser.ID)
	if ok {
		return fmt.Errorf("user %d already registered match server", waitingUser.ID)
	}

	s.MatchClients.Store(waitingUser.ID, conn)

	err := s.redisClient.AddUserToMatchQueue(waitingUser)
	if err != nil {
		return fmt.Errorf("failed to add user %d to queue: %v", waitingUser.ID, err)
	}

	log.Printf("User %d (gender: %d) added to waiting queue and MatchClients", waitingUser.ID, waitingUser.Gender)

	return nil
}

func (s *MatchService) UnregisterUserFromMatch(waitingUser commontype.WaitingUser) error {
	s.MatchClients.Delete(waitingUser.ID)

	if err := s.redisClient.RemoveUserFromQueue(waitingUser); err != nil {
		return fmt.Errorf("failed to remove user %d from queue: %v", waitingUser.ID, err)
	}

	log.Printf("User %d removed from waiting queue", waitingUser.ID)
	return nil
}

func (s *MatchService) SendMatchSuccessMessage(userIds []int, roomID string) {
	matchMsg := dto.MatchResponse{
		Type:   stype.PushMessageStatusMatchSuccess,
		RoomID: roomID,
	}

	payload, err := json.Marshal(matchMsg)
	if err != nil {
		log.Printf("Failed to marshal match response: %v", err)
		return
	}

	webSocketMsg := stype.WebSocketMessage{
		Kind:    stype.MessageTypeMatch,
		Payload: json.RawMessage(payload),
	}

	// TODO: 1명의 유저라도 실패할 경우 실패를 리턴하도록?
	for _, userID := range userIds {
		if conn, ok := s.MatchClients.Load(userID); ok {
			err := conn.(*websocket.Conn).WriteJSON(webSocketMsg)
			if err != nil {
				log.Printf("Failed to notify user %d: %v", userID, err)
			} else {
				s.MatchClients.Delete(userID)
			}
		} else {
			log.Printf("Failed to notify, user %d not connected", userID)
		}
	}
}

func (s *MatchService) SendMatchFailureMessage(conn *websocket.Conn) {
	matchMsg := dto.MatchResponse{
		Type:   stype.PushMessageStatusMatchFailure,
		RoomID: "",
	}

	payload, err := json.Marshal(matchMsg)
	if err != nil {
		log.Printf("Failed to marshal match response: %v", err)
		return
	}

	webSocketMsg := stype.WebSocketMessage{
		Kind:    stype.MessageTypeMatch,
		Payload: json.RawMessage(payload),
	}

	if err := conn.WriteJSON(webSocketMsg); err != nil {
		log.Printf("Failed to send match failure message: %v", err)
	}
}

// 매칭 큐 모니터링 및 이벤트 전송
func (s *MatchService) startMatchMonitoring() {
	log.Println("🔍 Starting match queue monitoring...")
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		for coupleCount := commontype.MATCH_COUNT_MIN; coupleCount <= commontype.MATCH_COUNT_MAX; coupleCount++ {
			matchedUsers, err := s.redisClient.MonitorAndMatchUsers(coupleCount)
			if err != nil {
				log.Printf("❌ Error while monitoring queue for %d: %v", coupleCount, err)
				continue
			}

			// 매칭된 사용자들 MQ로 이벤트 발행
			if len(matchedUsers) > 0 {
				matchedUsers := lo.Map(matchedUsers, func(user commontype.WaitingUser, _ int) commontype.MatchedUser {
					return commontype.MatchedUser{
						ID:     user.ID,
						Name:   user.Name,
						Gender: user.Gender,
						Birth:  user.Birth,
					}
				})
				s.notifyMatchSuccess(matchedUsers)
			}
		}
	}
}

// 매칭 성공 이벤트 MQ 발행
func (s *MatchService) notifyMatchSuccess(users []commontype.MatchedUser) {
	matchEvent := eventtypes.MatchEvent{
		MatchId:      generateMatchID(users),
		MatchType:    commontype.MATCH_GAME,
		MatchedUsers: users,
	}

	payload := eventtypes.EventPayload{
		EventType: eventtypes.EventTypeMatch,
		Data:      helper.ToJSON(matchEvent),
	}

	// MQ로 이벤트 전송
	err := s.emitter.PublishMatchEvent(payload)
	if err != nil {
		log.Printf("❌ Failed to publish match success event: %v", err)
	}

	logger.Info(logger.LogEventMatchSuccess, fmt.Sprintf("Match success: %s", matchEvent.MatchId), matchEvent)
}

func generateMatchID(users []commontype.MatchedUser) string {
	timestamp := time.Now().Format("20060102150405")
	var userIDs []string
	for _, user := range users {
		userIDs = append(userIDs, strconv.Itoa(user.ID))
	}
	return fmt.Sprintf("%s_%s", timestamp, joinIDs(userIDs))
}

func joinIDs(ids []string) string {
	return strings.Join(ids, "_")
}
