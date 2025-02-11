package service

import (
	"encoding/json"
	"fmt"
	"log"
	"solo/pkg/dto"
	"solo/pkg/mq"
	"solo/pkg/redis"
	"solo/pkg/types/commontype"
	"solo/pkg/types/sock"
	"sync"

	"github.com/gorilla/websocket"
)

type MatchService struct {
	redisClient  *redis.RedisClient
	mqClient     *mq.RabbitMQ
	MatchClients sync.Map
}

func NewMatchService(redisClient *redis.RedisClient, mqClient *mq.RabbitMQ) *MatchService {
	return &MatchService{
		redisClient: redisClient,
		mqClient:    mqClient,
	}
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
		Type:   sock.PushMessageStatusMatchSuccess,
		RoomID: roomID,
	}

	payload, err := json.Marshal(matchMsg)
	if err != nil {
		log.Printf("Failed to marshal match response: %v", err)
		return
	}

	webSocketMsg := sock.WebSocketMessage{
		Kind:    sock.MessageTypeMatch,
		Payload: json.RawMessage(payload),
	}

	// TODO: 1명의 유저라도 실패할 경우 실패를 리턴하도록?
	for _, userID := range userIds {
		log.Printf("[TEST] Try to notify user, %d", userID)

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
		Type:   sock.PushMessageStatusMatchFailure,
		RoomID: "",
	}

	payload, err := json.Marshal(matchMsg)
	if err != nil {
		log.Printf("Failed to marshal match response: %v", err)
		return
	}

	webSocketMsg := sock.WebSocketMessage{
		Kind:    sock.MessageTypeMatch,
		Payload: json.RawMessage(payload),
	}

	if err := conn.WriteJSON(webSocketMsg); err != nil {
		log.Printf("Failed to send match failure message: %v", err)
	}
}
