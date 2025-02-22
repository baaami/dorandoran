package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"solo/pkg/dto"
	"solo/pkg/redis"
	"solo/pkg/types/commontype"
	eventtypes "solo/pkg/types/eventtype"
	"solo/services/chat/repo"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MQEmitter interface {
	PublishChatRoomCreateEvent(data commontype.ChatRoom) error
	PublishCoupleRoomCreateEvent(data commontype.ChatRoom) error
	PublishChatLatestEvent(data eventtypes.ChatLatestEvent) error
	PublishRoomLeaveEvent(data eventtypes.RoomLeaveEvent) error
}

type ChatService struct {
	chatRepo    *repo.ChatRepository
	redisClient *redis.RedisClient
	emitter     MQEmitter
}

func NewChatService(chatRepo *repo.ChatRepository, redisClient *redis.RedisClient, emitter MQEmitter) *ChatService {
	return &ChatService{
		chatRepo:    chatRepo,
		redisClient: redisClient,
		emitter:     emitter,
	}
}

// 방 생성 (matchEvent 기반)
func (s *ChatService) CreateRoom(matchEvent eventtypes.MatchEvent) error {
	// 고유한 채팅방 ID 생성
	chatRoomID := matchEvent.MatchId

	var seq int64
	var roomName string
	var startTime time.Time
	var finishTime time.Time
	var gamers []commontype.GamerInfo

	if matchEvent.MatchType == commontype.MATCH_GAME {
		log.Printf("Create Game Room, users: %v", matchEvent.MatchedUsers)
		startTime = time.Now()
		finishTime = startTime.Add(20 * time.Second)

		seq, _ = s.chatRepo.GetNextSequence("chatRoomSeq")
		roomName = fmt.Sprintf("%d기", seq)
	} else {
		log.Printf("Create Couple Room, users: %v", matchEvent.MatchedUsers)
		startTime = time.Now()
		finishTime = startTime.Add(10 * time.Minute)

		seq = 0
		roomName = "커플 채팅방"
	}

	// 나는 솔로 캐릭터 할당
	male := 0
	female := 0

	for _, user := range matchEvent.MatchedUsers {
		var gamer commontype.GamerInfo

		if matchEvent.MatchType == commontype.MATCH_GAME {
			gamer.UserID = user.ID
			if user.Gender == commontype.MALE {
				gamer.CharacterID = male
				gamer.CharacterName = commontype.MaleNames[male]
				male++
			} else {
				gamer.CharacterID = female
				gamer.CharacterName = commontype.FemaleNames[female]
				female++
			}

			gamer.CharacterAvatarURL = fmt.Sprintf("/profile?gender=%d&character_id=%d", user.Gender, gamer.CharacterID)
		} else {
			// 커플 매칭일 경우 캐릭터 정보 없음
			gamer.CharacterID = -1
			gamer.CharacterAvatarURL = ""
		}

		gamers = append(gamers, gamer)
	}

	room := commontype.ChatRoom{
		ID:                  chatRoomID,
		Name:                roomName,
		Seq:                 seq,
		Type:                matchEvent.MatchType,
		UserIDs:             extractUserIDs(matchEvent.MatchedUsers),
		Gamers:              gamers,
		CreatedAt:           startTime,
		FinishChatAt:        finishTime,
		FinishFinalChoiceAt: finishTime.Add(30 * time.Second),
		ModifiedAt:          startTime,
	}

	// MongoDB에 채팅방 삽입
	err := s.chatRepo.InsertRoom(&room)
	if err != nil {
		log.Printf("Failed to insert chat room to MongoDB: %v", err)
		return err
	}

	// Redis에 채팅방 정보 추가
	err = s.redisClient.AddRoomToRedis(room.ID, room.UserIDs, time.Until(room.FinishChatAt))
	if err != nil {
		log.Printf("Failed to add room to Redis: %v", err)
		return err
	}

	// Redis에 채팅방 상태 설정
	err = s.redisClient.SetRoomStatus(room.ID, commontype.RoomStatusGameIng)
	if err != nil {
		log.Printf("Failed to set room status in Redis: %v", err)
		return err
	}

	// Redis에 타임아웃 설정
	err = s.redisClient.SetRoomTimeout(room.ID, time.Until(room.FinishChatAt))
	if err != nil {
		log.Printf("Failed to set room timeout in Redis: %v", err)
		return err
	}

	if matchEvent.MatchType == commontype.MATCH_GAME {
		err := s.emitter.PublishChatRoomCreateEvent(room)
		if err != nil {
			log.Printf("Failed to publish game room event: %v", err)
			return err
		}
	} else {
		err := s.emitter.PublishCoupleRoomCreateEvent(room)
		if err != nil {
			log.Printf("Failed to publish couple room event: %v", err)
			return err
		}
	}

	log.Printf("Chat room created: %s with users: %v", room.ID, room.UserIDs)
	return nil
}

// 매칭된 유저 ID 목록 추출
func extractUserIDs(users []commontype.WaitingUser) []int {
	ids := make([]int, len(users))
	for i, user := range users {
		ids[i] = user.ID
	}
	return ids
}

// 특정 유저가 속한 채팅방 목록 조회
func (s *ChatService) GetChatRoomList(userID int) ([]commontype.ChatRoom, error) {
	rooms, err := s.chatRepo.GetRoomsByUserID(userID)
	if err != nil {
		log.Printf("Failed to get chat rooms for user %d: %v", userID, err)
		return nil, err
	}
	return rooms, nil
}

func (s *ChatService) GetLatestMessage(roomID string) (*commontype.Chat, error) {
	return s.chatRepo.GetLastMessageByRoomID(roomID)
}

func (s *ChatService) GetUnreadCount(roomID string, userID int) (int, error) {
	return s.chatRepo.GetUnreadCountByUserAndRoom(userID, roomID)
}

func (s *ChatService) GetGamerInfo(userID int, roomID string) (*commontype.GamerInfo, error) {
	return s.chatRepo.GetUserGameInfoInRoom(userID, roomID)
}

// 특정 채팅방 상세 정보 조회
func (s *ChatService) GetChatRoomByID(roomID string) (*dto.RoomDetailResponse, error) {
	room, err := s.chatRepo.GetRoomByID(roomID)
	if err != nil {
		log.Printf("Failed to get chat room %s: %v", roomID, err)
		return nil, err
	}
	if room == nil {
		return nil, errors.New("chat room not found")
	}

	var gamerList []commontype.Gamer

	for _, userID := range room.UserIDs {
		user, err := getUserByUserID(strconv.Itoa(userID))
		if err != nil {
			log.Printf("Failed to get user %d: %v", userID, err)
		}

		gamer, err := s.chatRepo.GetUserGameInfoInRoom(userID, room.ID)
		if err != nil {
			log.Printf("Failed to get user game info %d: %v", userID, err)
		}

		gamerList = append(gamerList, commontype.Gamer{
			ID:      user.ID,
			SnsType: user.SnsType,
			SnsID:   user.SnsID,
			Name:    user.Name,
			Gender:  user.Gender,
			Birth:   user.Birth,
			Address: user.Address,
			GameInfo: commontype.GameInfo{
				CharacterID:        gamer.CharacterID,
				CharacterName:      gamer.CharacterName,
				CharacterAvatarURL: gamer.CharacterAvatarURL,
			},
		})
	}

	roomDetail := dto.RoomDetailResponse{
		ID:                  room.ID,
		Type:                room.Type,
		Status:              room.Status,
		Seq:                 int(room.Seq),
		RoomName:            room.Name,
		Users:               gamerList,
		CreatedAt:           room.CreatedAt,
		FinishChatAt:        room.FinishChatAt,
		FinishFinalChoiceAt: room.FinishFinalChoiceAt,
	}

	return &roomDetail, nil
}

// 채팅방 참여 처리
func (s *ChatService) HandleRoomJoin(roomID string, userID int, joinTime time.Time) error {
	room, err := s.chatRepo.GetRoomByID(roomID)
	if err != nil {
		return err
	}
	if room == nil {
		return errors.New("chat room not found")
	}

	// 읽지 않은 메시지 처리
	messages, err := s.chatRepo.GetUnreadMessagesBefore(roomID, joinTime, userID)
	if err != nil {
		return err
	}

	var messageIDs []primitive.ObjectID
	for _, message := range messages {
		reader := commontype.ChatReader{
			MessageId: message.MessageId,
			RoomID:    roomID,
			UserId:    userID,
			ReadAt:    joinTime,
		}
		if err := s.chatRepo.InsertChatReader(reader); err != nil {
			return err
		}

		messageIDs = append(messageIDs, message.MessageId)
	}

	if len(messageIDs) > 0 {
		err := s.chatRepo.UpdateUnreadCounts(messageIDs)
		if err != nil {
			log.Printf("Failed to update unread counts, roomid %s, err: %v", roomID, err)
		}

		err = s.emitter.PublishChatLatestEvent(eventtypes.ChatLatestEvent{
			RoomID: roomID,
		})
		if err != nil {
			log.Printf("Failed to PublishChatLatestEvent, roomid %s, err: %v", roomID, err)
		}
	}

	return nil
}

// 채팅방 나가기
func (s *ChatService) LeaveChatRoom(roomID string, userID int) error {
	return s.chatRepo.LeaveRoom(roomID, userID)
}

// 채팅방 삭제
func (s *ChatService) DeleteChatRoom(roomID string) error {
	return s.chatRepo.DeleteRoom(roomID)
}

// 채팅 메시지 추가
func (s *ChatService) AddChatMsg(chatMsg commontype.Chat) (primitive.ObjectID, error) {
	messageID, err := s.chatRepo.InsertChatMessage(chatMsg)
	if err != nil {
		log.Printf("Failed to insert chat message: %v", err)
		return primitive.NilObjectID, err
	}
	return messageID, nil
}

// 채팅 메시지 읽음 처리
func (s *ChatService) HandleChatRead(messageID primitive.ObjectID, roomID string, readerIDs []int, readAt time.Time) error {
	for _, userID := range readerIDs {
		reader := commontype.ChatReader{
			MessageId: messageID,
			RoomID:    roomID,
			UserId:    userID,
			ReadAt:    readAt,
		}

		err := s.chatRepo.InsertChatReader(reader)
		if err != nil {
			log.Printf("Failed to insert ChatReader for user %d: %v", userID, err)
			return err
		}
	}

	return s.chatRepo.UpdateUnreadCounts([]primitive.ObjectID{messageID})
}

// 특정 채팅방의 메시지 목록 조회 (페이징 포함)
func (s *ChatService) GetChatMsgListByRoomID(roomID string, pageNumber int, pageSize int) ([]*commontype.Chat, int64, error) {
	messages, totalCount, err := s.chatRepo.GetByRoomIDWithPagination(roomID, pageNumber, pageSize)
	if err != nil {
		log.Printf("Failed to get chat messages for room %s: %v", roomID, err)
		return nil, 0, err
	}
	return messages, totalCount, nil
}

func (s *ChatService) UpdateChatRoomStatus(roomID string, status int) error {
	return s.chatRepo.UpdateChatRoomStatus(roomID, status)
}

// 특정 채팅방의 메시지 삭제
func (s *ChatService) DeleteChatByRoomID(roomID string) error {
	return s.chatRepo.DeleteChatByRoomID(roomID)
}

// 특정 유저의 게임 캐릭터 정보 조회
func (s *ChatService) GetCharacterNameByRoomID(userID int, roomID string) (*commontype.GamerInfo, error) {
	return s.chatRepo.GetUserGameInfoInRoom(userID, roomID)
}

// [Bridge user] 회원 정보 획득
func getUserByUserID(userID string) (*commontype.User, error) {
	client := &http.Client{
		Timeout: time.Second * 10, // 요청 타임아웃 설정
	}

	// 요청 URL 생성
	url := "http://user-service/find"

	// GET 요청 생성
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// 사용자 ID를 요청의 헤더에 추가
	req.Header.Set("X-User-ID", userID)

	// 요청 실행
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// 응답 처리
	if resp.StatusCode == http.StatusNotFound {
		// 유저가 존재하지 않는 경우
		return nil, nil
	} else if resp.StatusCode != http.StatusOK {
		// 다른 에러가 발생한 경우
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// 응답 본문에서 유저 정보 디코딩
	var user commontype.User

	// 응답 본문 로깅 추가
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// 본문을 다시 디코딩
	err = json.Unmarshal(body, &user)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// 유저가 존재하는 경우
	return &user, nil
}
