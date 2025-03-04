package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"solo/pkg/dto"
	"solo/pkg/helper"
	"solo/pkg/redis"
	"solo/pkg/types/commontype"
	eventtypes "solo/pkg/types/eventtype"
	"solo/pkg/types/stype"
	"solo/services/chat/repo"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MQEmitter interface {
	PublishRoomJoinEvent(data eventtypes.RoomJoinEvent) error
	PublishChatMessageEvent(event eventtypes.ChatEvent) error
	PublishFinalChoiceTimeoutEvent(event eventtypes.FinalChoiceTimeoutEvent) error
	PublishRoomTimeoutEvent(timeoutEvent eventtypes.RoomTimeoutEvent) error
}

// Client 구조체 - WebSocket 클라이언트
type Client struct {
	Conn *websocket.Conn
	Send chan interface{}
	Ctx  context.Context
}

// GameService - 게임 서비스 계층
type GameService struct {
	redisClient *redis.RedisClient
	chatRepo    *repo.ChatRepository
	clients     sync.Map // key: userID, value: *Client
	emitter     MQEmitter
}

// NewGameService - GameService 인스턴스 생성
func NewGameService(redisClient *redis.RedisClient, emitter MQEmitter, chatRepo *repo.ChatRepository) *GameService {
	service := &GameService{
		redisClient: redisClient,
		chatRepo:    chatRepo,
		emitter:     emitter,
	}

	// 게임방 대화 시간 타임아웃 모니터링
	go service.MonitorChatTimeouts()

	// 최종 선택 시간 타임아웃 모니터링
	go service.MonitorFinalChoiceTimeouts()

	return service
}

// RegisterUserToGame - 사용자를 게임에 등록하고 Redis 활성화
func (s *GameService) RegisterUserToGame(userID int, client *Client) error {
	// WebSocket 클라이언트 저장
	s.clients.Store(userID, client)

	// Redis에 활성 사용자 등록
	serverID := commontype.DEFAULT_TEMP_SERVER_ID // TODO: 서버 고유 ID 설정 필요
	err := s.redisClient.RegisterActiveUser(userID, serverID)
	if err != nil {
		log.Printf("❌ Redis 사용자 등록 실패: %v", err)
		return err
	}

	log.Printf("✅ 사용자 게임 등록: User %d", userID)
	return nil
}

// UnRegisterUserFromGame - 사용자를 게임에서 제거하고 Redis에서 삭제
func (s *GameService) UnRegisterUserFromGame(userID int) {
	// WebSocket 클라이언트 제거
	if clientInterface, ok := s.clients.Load(userID); ok {
		client := clientInterface.(*Client)
		close(client.Send) // Send 채널 닫기
		s.clients.Delete(userID)
	}

	// Redis에서 활성 사용자 제거
	err := s.redisClient.UnregisterActiveUser(userID)
	if err != nil {
		log.Printf("❌ Redis 사용자 제거 실패: %v", err)
	} else {
		log.Printf("✅ 사용자 게임 제거: User %d", userID)
	}
}

func (s *GameService) BroadcastMessage(roomID string, userID int, message string, headCnt int) error {
	log.Printf("💬 User %d sending message to room %s", userID, roomID)

	// Redis에서 비활성 사용자 목록 조회
	inactiveUserIDs, err := s.redisClient.GetInActiveUserIDs(roomID)
	if err != nil {
		return fmt.Errorf("❌ Redis GetInActiveUserIDs 실패: %w", err)
	}

	// 방에 접속해있는 사용자 ID 리스트 가져오기
	joinedUserIDs, err := s.redisClient.GetJoinedUser(roomID)
	if err != nil {
		return fmt.Errorf("❌ Redis GetJoinedUser 실패: %w", err)
	}

	// 메시지 생성
	chatEvent := eventtypes.ChatEvent{
		MessageId:       primitive.NewObjectID(),
		Type:            commontype.ChatTypeChat,
		RoomID:          roomID,
		SenderID:        userID,
		Message:         message,
		UnreadCount:     headCnt - len(joinedUserIDs),
		InactiveUserIds: inactiveUserIDs,
		ReaderIds:       joinedUserIDs,
		CreatedAt:       time.Now(),
	}

	// RabbitMQ를 통해 메시지 전송
	err = s.emitter.PublishChatMessageEvent(chatEvent)
	if err != nil {
		log.Printf("⚠️ RabbitMQ PublishChatMessageEvent 실패: %v", err)
	}

	return nil
}

func (s *GameService) SendMessageToRoom(roomID string, message stype.WebSocketMessage) error {
	activeUserIDs, err := s.redisClient.GetActiveUserIDs(roomID)
	if err != nil {
		log.Printf("❌ Redis GetActiveUserIDs 실패: %v", err)
		return err
	}

	for _, userID := range activeUserIDs {
		if client, ok := s.clients.Load(userID); ok {
			log.Printf("📨 Sending WebSocket %s message to User %d in Room %s", message.Kind, userID, roomID)

			client.(*Client).Send <- message
		}
	}

	return nil
}

func (s *GameService) SendMessageToUser(userID int, message stype.WebSocketMessage) error {
	if client, ok := s.clients.Load(userID); ok {
		log.Printf("📨 Sending WebSocket %s message to User %d", message.Kind, userID)

		client.(*Client).Send <- message
	}

	return nil
}

func (s *GameService) JoinGameRoom(roomID string, userID int) error {
	log.Printf("🎮 User %d joining game room %s", userID, roomID)

	// Redis에 게임방 참가 정보 저장
	err := s.redisClient.JoinRoom(roomID, userID)
	if err != nil {
		return fmt.Errorf("❌ Redis JoinRoom 실패: %w", err)
	}

	// RabbitMQ를 통해 이벤트 발행
	roomJoinMsg := eventtypes.RoomJoinEvent{
		RoomID: roomID,
		UserID: userID,
		JoinAt: time.Now(),
	}

	err = s.emitter.PublishRoomJoinEvent(roomJoinMsg)
	if err != nil {
		log.Printf("⚠️ RabbitMQ PublishRoomJoinEvent 실패: %v", err)
	}

	return nil
}

func (s *GameService) LeaveGameRoom(roomID string, userID int) error {
	log.Printf("🚪 User %d leaving game room %s", userID, roomID)

	// Redis에서 유저 제거
	err := s.redisClient.LeaveRoom(roomID, userID)
	if err != nil {
		return fmt.Errorf("❌ Redis LeaveRoom 실패: %w", err)
	}

	return nil
}

func (s *GameService) BroadCastFinalChoiceStart(roomID string) error {
	// 현재 룸 상태 확인
	status, err := s.redisClient.GetRoomStatus(roomID)
	if err != nil {
		return fmt.Errorf("❌ Redis GetRoomStatus 실패: %w", err)
	}

	room, err := s.chatRepo.GetRoomByID(roomID)
	if err != nil {
		return fmt.Errorf("❌ Redis GetRoomByID 실패: %w", err)
	}

	if status != commontype.RoomStatusGameIng {
		log.Printf("⚠️ Room %s is not in active game state, skipping timeout process.", roomID)
		return nil
	}

	payload, err := json.Marshal(stype.FinalChoiceStartMessage{RoomID: roomID, RoomName: room.Name})
	if err != nil {
		return fmt.Errorf("❌ FinalChoiceStartMessage 직렬화 실패: %w", err)
	}

	message := stype.WebSocketMessage{
		Kind:    stype.MessageKindFinalChoiceStart,
		Payload: json.RawMessage(payload),
	}

	err = s.SendMessageToRoom(roomID, message)
	if err != nil {
		return fmt.Errorf("❌ WebSocket final_choice_start 전송 실패: %w", err)
	}

	// Redis에서 타임아웃 데이터 정리
	err = s.redisClient.ClearFinalChoiceRoom(roomID)
	if err != nil {
		log.Printf("❌ Redis ClearFinalChoiceRoom 실패: %v", err)
	}

	err = s.redisClient.SetRoomStatus(roomID, commontype.RoomStatusChoiceIng)
	if err != nil {
		return fmt.Errorf("❌ Redis SetRoomStatus(RoomStatusChoiceIng) 실패: %w", err)
	}

	roomDetail, err := GetRoomDetail(roomID)
	if err != nil {
		return fmt.Errorf("❌ Redis GetRoomDetail 실패: %w", err)
	}

	err = s.redisClient.SetFinalChoiceTimeout(roomID, time.Until(roomDetail.FinishFinalChoiceAt))
	if err != nil {
		return fmt.Errorf("❌ Redis SetFinalChoiceTimeout 실패: %w", err)
	}

	return nil
}

func (s *GameService) ProcessRoomTimeoutMessage(roomTimeoutMsg stype.RoomTimeoutMessage, userID int) error {
	err := s.redisClient.AddChatTimeoutUser(roomTimeoutMsg.RoomID, userID)
	if err != nil {
		log.Printf("Failed to SaveUserChoice, err: %v", err)
		return nil
	}

	roomTimeoutUserIds, err := s.redisClient.GetChatTimeoutUserCount(roomTimeoutMsg.RoomID)
	if err != nil {
		log.Printf("Failed to GetChatTimeoutUserCount, err: %v", err)
		return nil
	}

	roomTotalUserIds, err := s.redisClient.GetRoomUserIDs(roomTimeoutMsg.RoomID)
	if err != nil {
		log.Printf("Failed to GetRoomUserIDs, err: %v", err)
		return nil
	}

	if int(roomTimeoutUserIds) == len(roomTotalUserIds) {
		s.BroadCastFinalChoiceStart(roomTimeoutMsg.RoomID)
	}

	return nil
}

func (s *GameService) BroadcastFinalChoices(roomID string) error {
	log.Printf("📢 Broadcasting final choices for Room %s", roomID)

	// Redis에서 최종 선택 결과 조회
	finalChoiceResults, err := s.redisClient.GetAllChoices(roomID)
	if err != nil {
		return fmt.Errorf("❌ Redis GetAllChoices 실패: %w", err)
	}

	// JSON 직렬화
	payload, err := json.Marshal(finalChoiceResults)
	if err != nil {
		return fmt.Errorf("❌ Final choices 직렬화 실패: %w", err)
	}

	// WebSocket 메시지 생성
	message := stype.WebSocketMessage{
		Kind:    stype.MessageKindFinalChoiceResult,
		Payload: json.RawMessage(payload),
	}

	// 활성 유저에게 최종 선택 결과 전송
	err = s.SendMessageToRoom(roomID, message)
	if err != nil {
		return fmt.Errorf("❌ WebSocket 전송 실패: %w", err)
	}

	log.Printf("✅ Final choices broadcasted to Room %s", roomID)

	// Redis에서 최종 선택 정보 삭제
	err = s.redisClient.ClearFinalChoiceRoom(roomID)
	if err != nil {
		return fmt.Errorf("❌ Redis ClearFinalChoiceRoom 실패: %w", err)
	}

	return nil
}

func (s *GameService) ProcessFinalChoice(userID int, finalChoiceMsg stype.FinalChoiceMessage) error {
	roomID := finalChoiceMsg.RoomID
	selectedUserID := finalChoiceMsg.SelectedUserID
	log.Printf("💘 User %d selected User %d in Room %s", userID, selectedUserID, roomID)

	// 유저의 선택을 Redis에 저장
	err := s.redisClient.SaveUserChoice(roomID, userID, selectedUserID)
	if err != nil {
		return fmt.Errorf("❌ Redis SaveUserChoice 실패: %w", err)
	}

	// 방에 참여한 전체 유저 수 조회
	roomTotalUserIDs, err := s.redisClient.GetRoomUserIDs(roomID)
	if err != nil {
		return fmt.Errorf("❌ Redis GetRoomUserIDs 실패: %w", err)
	}

	// 모든 유저가 선택을 완료했는지 확인
	allChosen, err := s.redisClient.IsAllChoicesCompleted(roomID, int64(len(roomTotalUserIDs)))
	if err != nil {
		return fmt.Errorf("❌ Redis IsAllChoicesCompleted 실패: %w", err)
	}

	if allChosen {
		log.Printf("🎉 All users in Room %s have made their final choices!", roomID)

		// 최종 선택 결과 전송
		err = s.BroadcastFinalChoices(roomID)
		if err != nil {
			return fmt.Errorf("❌ BroadcastFinalChoices 실패: %w", err)
		}
	}

	return nil
}

// TODO: Bridige network 사용하지 않도록!!
func GetRoomDetail(roomID string) (*dto.RoomDetailResponse, error) {
	var chatRoomDetail dto.RoomDetailResponse

	// Matching 필터 획득
	client := http.Client{}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://doran-chat/room/%s", roomID), nil)
	if err != nil {
		return nil, err
	}

	// 요청 실행
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	err = json.Unmarshal(body, &chatRoomDetail)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &chatRoomDetail, nil
}

// 특정 채팅방 타임아웃 감지
func (s *GameService) MonitorChatTimeouts() {
	ticker := time.NewTicker(3 * time.Second) // 최대 1초 내에 이벤트 감지
	defer ticker.Stop()

	for range ticker.C {
		// Redis에 저장된 모든 방 ID 가져오기
		rooms, err := s.redisClient.GetAllRoomsFromRedis()
		if err != nil {
			log.Printf("Failed to fetch rooms for timeout monitoring: %v", err)
			continue
		}

		for _, roomID := range rooms {
			// 남은 시간이 0 이하인지 확인
			remainingTime, err := s.redisClient.GetRoomRemainingTime(roomID)
			if err != nil || remainingTime > 0 {
				continue // 아직 만료되지 않은 방은 스킵
			}

			inactiveUsers, err := s.redisClient.GetInActiveUserIDs(roomID)
			if err != nil {
				log.Printf("Failed to get inactive users, err: %s", err.Error())
				continue
			}

			event := eventtypes.RoomTimeoutEvent{
				RoomID:          roomID,
				InactiveUserIds: inactiveUsers,
			}

			// 만료된 방에 대해 timeout 이벤트 발행
			err = s.emitter.PublishRoomTimeoutEvent(event)
			if err != nil {
				log.Printf("Failed to handle timeout for RoomID %s: %v", roomID, err)
			}

			// TODO: Redis에서 최종 선택 완료 시 방 삭제
			err = s.redisClient.RemoveRoomFromRedis(roomID)
			if err != nil {
				log.Printf("Failed to remove expired room %s from Redis: %v", roomID, err)
			}
		}
	}
}

func (s *GameService) MonitorFinalChoiceTimeouts() {
	ticker := time.NewTicker(3 * time.Second) // 최대 1초 내에 이벤트 감지
	defer ticker.Stop()

	for range ticker.C {
		// Redis에 저장된 모든 방 ID 가져오기
		rooms, err := s.redisClient.GetAllChoiceRoomsFromRedis()
		if err != nil {
			log.Printf("Failed to GetAllChoiceRoomsFromRedis, err: %v", err)
			continue
		}

		for _, roomID := range rooms {
			// 남은 시간이 0 이하인지 확인
			remainingTime, err := s.redisClient.GetChoiceRoomRemainingTime(roomID)
			if err != nil || remainingTime > 0 {
				continue // 아직 만료되지 않은 방은 스킵
			}

			userIds, err := s.redisClient.GetRoomUserIDs(roomID)
			if err != nil {
				log.Printf("Failed to GetRoomUserIDs, room: %s, err: %v", roomID, err)
				continue
			}

			roomTotalUserIds, err := helper.StringToIntArrary(userIds)
			if err != nil {
				log.Printf("Failed to ConvertStringSliceToIntSlice, room: %s, err: %v", roomID, err)
				continue
			}

			event := eventtypes.FinalChoiceTimeoutEvent{
				RoomID:  roomID,
				UserIDs: roomTotalUserIds,
			}

			// 만료된 방에 대해 timeout 이벤트 발행
			err = s.emitter.PublishFinalChoiceTimeoutEvent(event)
			if err != nil {
				log.Printf("Failed to handle timeout for RoomID %s: %v", roomID, err)
			}

			err = s.redisClient.RemoveChoiceRoomFromRedis(roomID)
			if err != nil {
				log.Printf("Failed to remove expired room %s from Redis: %v", roomID, err)
			}
		}
	}
}
