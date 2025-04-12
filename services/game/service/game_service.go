package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"solo/pkg/dto"
	"solo/pkg/helper"
	"solo/pkg/logger"
	"solo/pkg/models"
	"solo/pkg/redis"
	"solo/pkg/types/commontype"
	eventtypes "solo/pkg/types/eventtype"
	"solo/pkg/utils/stype"

	"solo/services/chat/repo"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MQEmitter interface {
	PublishRoomJoinEvent(data eventtypes.RoomJoinEvent) error
	PublishChatMessageEvent(event eventtypes.ChatEvent) error
	PublishFinalChoiceTimeoutEvent(event eventtypes.FinalChoiceTimeoutEvent) error
	PublishRoomTimeoutEvent(timeoutEvent eventtypes.RoomTimeoutEvent) error
	PublishMatchEvent(event eventtypes.MatchEvent) error
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

	// 밸런스 게임 타이머 모니터링
	go service.MonitorBalanceGameStartTimer()

	// 밸런스 게임 종료 모니터링
	go service.MonitorBalanceGameFinishTimer()

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

	err = s.redisClient.SetFinalChoiceTimeout(roomID, time.Until(room.FinishFinalChoiceAt))
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

	chatRoom, err := s.chatRepo.GetRoomByID(roomID)
	if err != nil {
		return fmt.Errorf("❌ GetRoomByID 실패: %w", err)
	}

	matchStrings := helper.ConvertUserChoicesToMatchStrings(finalChoiceResults.Choices)

	err = s.sendCoupleMatchEvent(matchStrings)
	if err != nil {
		return fmt.Errorf("❌ sendCoupleMatchEvent 실패: %w", err)
	}

	// 최종 선택 결과 저장
	err = s.chatRepo.UpdateMatchHistoryFinalMatch(int(chatRoom.Seq), matchStrings)
	if err != nil {
		return fmt.Errorf("❌ UpdateFinalMatch 실패: %w", err)
	}

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

// 채팅 시간 타임아웃 모니터링
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

			err = s.chatRepo.UpdateChatRoomStatus(roomID, commontype.RoomStatusChoiceIng)
			if err != nil {
				log.Printf("Failed to update chat room status, err: %v", err)
			}

			// 만료된 방에 대해 timeout 이벤트 발행
			err = s.emitter.PublishRoomTimeoutEvent(event)
			if err != nil {
				log.Printf("Failed to handle timeout for RoomID %s: %v", roomID, err)
			}

			logger.Info(logger.LogEventGameRoomChatTimeout, fmt.Sprintf("Chat room timeout: %s", roomID), event)

			// TODO: Redis에서 최종 선택 완료 시 방 삭제
			err = s.redisClient.RemoveRoomFromRedis(roomID)
			if err != nil {
				log.Printf("Failed to remove expired room %s from Redis: %v", roomID, err)
			}
		}
	}
}

// 최종 선택 시간 타임아웃 모니터링
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

			err = s.chatRepo.UpdateChatRoomStatus(roomID, commontype.RoomStatusGameEnd)
			if err != nil {
				log.Printf("Failed to update chat room status, err: %v", err)
			}

			// 만료된 방에 대해 timeout 이벤트 발행
			err = s.emitter.PublishFinalChoiceTimeoutEvent(event)
			if err != nil {
				log.Printf("Failed to handle timeout for RoomID %s: %v", roomID, err)
			}

			logger.Info(logger.LogEventFinalChoiceEnd, fmt.Sprintf("Final choice timeout: %s", roomID), event)

			err = s.redisClient.RemoveChoiceRoomFromRedis(roomID)
			if err != nil {
				log.Printf("Failed to remove expired room %s from Redis: %v", roomID, err)
			}
		}
	}
}

// 밸런스 게임 타이머 모니터링
func (s *GameService) MonitorBalanceGameStartTimer() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Redis에서 밸런스 게임 타이머가 설정된 모든 방 가져오기
		rooms, err := s.redisClient.GetAllBalanceGameRooms()
		if err != nil {
			log.Printf("Failed to get balance game rooms: %v", err)
			continue
		}

		for _, roomID := range rooms {
			// 남은 시간 확인
			remainingTime, err := s.redisClient.GetBalanceGameRemainingTime(roomID)
			if err != nil || remainingTime > 0 {
				continue
			}

			// 밸런스 게임 랜덤 획득
			balanceGame, err := s.chatRepo.GetRandomBalanceGameForm()
			if err != nil {
				log.Printf("Failed to get random balance game form: %v", err)
				continue
			}

			// 시간이 다 되었으면 밸런스 게임 시작 메시지 전송
			balanceGameForm := &models.BalanceGameForm{
				Question: models.Question{
					Title: balanceGame.Title,
					Red:   balanceGame.Red,
					Blue:  balanceGame.Blue,
				},
				RoomID: roomID,
			}

			// 밸런스 게임 폼 저장
			formID, err := s.chatRepo.InsertBalanceForm(balanceGameForm)
			if err != nil {
				log.Printf("Failed to insert balance game form: %v", err)
				continue
			}

			// Redis에서 비활성 사용자 목록 조회
			inactiveUserIDs, err := s.redisClient.GetInActiveUserIDs(roomID)
			if err != nil {
				log.Printf("Failed to GetInActiveUserIDs, room: %s, err: %v", roomID, err)
				continue
			}

			// 방에 접속해있는 사용자 ID 리스트 가져오기
			joinedUserIDs, err := s.redisClient.GetJoinedUser(roomID)
			if err != nil {
				log.Printf("Failed to GetJoinedUser, room: %s, err: %v", roomID, err)
				continue
			}

			headCnt, err := s.redisClient.GetRoomUserIDs(roomID)
			if err != nil {
				log.Printf("Failed to GetRoomUserIDs, room: %s, err: %v", roomID, err)
				continue
			}

			// 채팅 메시지 생성
			chatEvent := eventtypes.ChatEvent{
				MessageId:       primitive.NewObjectID(),
				Type:            commontype.ChatTypeForm,
				RoomID:          roomID,
				SenderID:        commontype.MasterID,
				Message:         "밸런스 게임을 시작합니다!",
				BalanceFormID:   formID,
				UnreadCount:     len(headCnt) - len(joinedUserIDs),
				InactiveUserIds: inactiveUserIDs,
				ReaderIds:       joinedUserIDs,
				CreatedAt:       time.Now(),
			}

			// RabbitMQ를 통해 메시지 전송
			err = s.emitter.PublishChatMessageEvent(chatEvent)
			if err != nil {
				log.Printf("Failed to publish balance game start message: %v", err)
			}

			// Redis에서 해당 방의 밸런스 게임 타이머 제거
			err = s.redisClient.RemoveBalanceGameRoom(roomID)
			if err != nil {
				log.Printf("Failed to remove balance game timer: %v", err)
			}

			// 밸런스 게임 종료 타이머 설정 (15분)
			err = s.redisClient.SetBalanceGameFinishTimer(formID.Hex(), commontype.BalanceGameEndTimer)
			if err != nil {
				log.Printf("Failed to set balance game finish timer: %v", err)
			}

			logger.Info(logger.LogEventBalanceGameStart, fmt.Sprintf("Balance game start: %s", roomID), chatEvent)
		}
	}
}

// 밸런스 게임 종료 모니터링
func (s *GameService) MonitorBalanceGameFinishTimer() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Redis에서 밸런스 게임 종료 타이머가 설정된 모든 form 가져오기
		forms, err := s.redisClient.GetAllBalanceGameFinishForms()
		if err != nil {
			log.Printf("Failed to get balance game finish forms: %v", err)
			continue
		}

		for _, formID := range forms {
			// 남은 시간 확인
			remainingTime, err := s.redisClient.GetBalanceGameFinishRemainingTime(formID)
			if err != nil || remainingTime > 0 {
				continue
			}

			// form ID를 ObjectID로 변환
			formObjectID, err := primitive.ObjectIDFromHex(formID)
			if err != nil {
				log.Printf("Failed to convert form ID to ObjectID: %v", err)
				continue
			}

			// form 정보 조회
			form, err := s.chatRepo.GetBalanceFormByID(formObjectID)
			if err != nil {
				log.Printf("Failed to get balance form: %v", err)
				continue
			}

			roomID := form.RoomID

			// Redis에서 비활성 사용자 목록 조회
			inactiveUserIDs, err := s.redisClient.GetInActiveUserIDs(roomID)
			if err != nil {
				log.Printf("Failed to GetInActiveUserIDs, room: %s, err: %v", roomID, err)
				continue
			}

			// 방에 접속해있는 사용자 ID 리스트 가져오기
			joinedUserIDs, err := s.redisClient.GetJoinedUser(roomID)
			if err != nil {
				log.Printf("Failed to GetJoinedUser, room: %s, err: %v", roomID, err)
				continue
			}

			headCnt, err := s.redisClient.GetRoomUserIDs(roomID)
			if err != nil {
				log.Printf("Failed to GetRoomUserIDs, room: %s, err: %v", roomID, err)
				continue
			}

			// 채팅 메시지 생성
			chatEvent := eventtypes.ChatEvent{
				MessageId:       primitive.NewObjectID(),
				Type:            commontype.ChatTypeFormResult,
				RoomID:          roomID,
				SenderID:        commontype.MasterID,
				Message:         "밸런스 게임이 종료되었습니다!",
				BalanceFormID:   form.ID,
				UnreadCount:     len(headCnt) - len(joinedUserIDs),
				InactiveUserIds: inactiveUserIDs,
				ReaderIds:       joinedUserIDs,
				CreatedAt:       time.Now(),
			}

			// RabbitMQ를 통해 메시지 전송
			err = s.emitter.PublishChatMessageEvent(chatEvent)
			if err != nil {
				log.Printf("Failed to publish balance game finish message: %v", err)
			}

			log.Printf("🎮 Balance game in room %s has finished! Form ID: %s", form.RoomID, formID)

			// Redis에서 해당 form의 밸런스 게임 종료 타이머 제거
			err = s.redisClient.RemoveBalanceGameFinishForm(formID)
			if err != nil {
				log.Printf("Failed to remove balance game finish timer: %v", err)
			}

			logger.Info(logger.LogEventBalanceGameEnd, fmt.Sprintf("Balance game end: %s", roomID), chatEvent)
		}
	}
}

func (s *GameService) sendCoupleMatchEvent(matchStrings []string) error {

	log.Printf("🔍 matchStrings: %v", matchStrings)

	// matchStrings 분석하여 매칭된 사용자 정보 추출
	for _, matchStr := range matchStrings {
		var matchedUsers []commontype.MatchedUser

		// 매칭 문자열 파싱 (예: "1:2")
		users := strings.Split(matchStr, ":")
		if len(users) != 2 {
			continue
		}

		user1ID, err := strconv.Atoi(users[0])
		if err != nil {
			continue
		}

		user2ID, err := strconv.Atoi(users[1])
		if err != nil {
			continue
		}

		user1, err := GetUserInfo(user1ID)
		if err != nil {
			continue
		}

		user2, err := GetUserInfo(user2ID)
		if err != nil {
			continue
		}

		// 매칭된 사용자 정보 추가
		matchedUsers = append(matchedUsers, commontype.MatchedUser{
			ID:     user1ID,
			Name:   user1.Name,
			Gender: user1.Gender,
			Birth:  user1.Birth,
		})

		matchedUsers = append(matchedUsers, commontype.MatchedUser{
			ID:     user2ID,
			Name:   user2.Name,
			Gender: user2.Gender,
			Birth:  user2.Birth,
		})

		// 매칭 이벤트 생성 및 발행
		matchEvent := eventtypes.MatchEvent{
			MatchId:      helper.GenerateMatchID(matchedUsers),
			MatchType:    commontype.MATCH_COUPLE,
			MatchedUsers: matchedUsers,
		}

		log.Printf("🔍 matchEvent: %v", matchEvent)

		err = s.emitter.PublishMatchEvent(matchEvent)
		if err != nil {
			return fmt.Errorf("❌ PublishMatchEvent 실패: %w", err)
		}
	}

	return nil
}

func GetUserInfo(userID int) (commontype.MatchedUser, error) {
	// User 서비스 API 엔드포인트 설정
	userServiceURL := "http://doran-user/find"
	client := &http.Client{}

	// 요청 생성
	req, err := http.NewRequest("GET", userServiceURL, nil)
	if err != nil {
		return commontype.MatchedUser{}, fmt.Errorf("❌ 요청 생성 실패: %w", err)
	}

	// X-User-ID 헤더 설정
	req.Header.Set("X-User-ID", strconv.Itoa(userID))

	// 요청 실행
	resp, err := client.Do(req)
	if err != nil {
		return commontype.MatchedUser{}, fmt.Errorf("❌ API 요청 실패: %w", err)
	}
	defer resp.Body.Close()

	// 응답 상태 코드 확인
	if resp.StatusCode != http.StatusOK {
		return commontype.MatchedUser{}, fmt.Errorf("❌ API 응답 오류: %d", resp.StatusCode)
	}

	// 응답 데이터 파싱
	var user dto.UserDTO
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return commontype.MatchedUser{}, fmt.Errorf("❌ 응답 데이터 파싱 실패: %w", err)
	}

	// MatchedUser 객체로 변환
	matchedUser := commontype.MatchedUser{
		ID:     user.ID,
		Name:   user.Name,
		Gender: user.Gender,
		Birth:  user.Birth,
	}

	return matchedUser, nil
}
