package service

import (
	"errors"
	"fmt"
	"log"
	"time"

	"solo/pkg/dto"
	"solo/pkg/models"
	"solo/pkg/redis"
	"solo/pkg/types/commontype"
	eventtypes "solo/pkg/types/eventtype"
	"solo/services/chat/repo"
	"solo/services/user/repository"

	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MQEmitter interface {
	PublishChatRoomCreateEvent(data models.ChatRoom) error
	PublishCoupleRoomCreateEvent(data models.ChatRoom) error
	PublishChatLatestEvent(data eventtypes.ChatLatestEvent) error
	PublishRoomLeaveEvent(data eventtypes.RoomLeaveEvent) error
	PublishVoteCommentChatEvent(event eventtypes.VoteCommentChatEvent) error
}

type ChatService struct {
	chatRepo    *repo.ChatRepository
	userRepo    *repository.UserRepository
	redisClient *redis.RedisClient
	emitter     MQEmitter
}

func NewChatService(chatRepo *repo.ChatRepository, userRepo *repository.UserRepository, redisClient *redis.RedisClient, emitter MQEmitter) *ChatService {
	return &ChatService{
		chatRepo:    chatRepo,
		userRepo:    userRepo,
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
	var gamers []models.GamerInfo

	if matchEvent.MatchType == commontype.MATCH_GAME {
		log.Printf("Create Game Room, users: %v", matchEvent.MatchedUsers)
		startTime = time.Now()
		finishTime = startTime.Add(commontype.GameRunningTime)

		seq, _ = s.chatRepo.GetNextSequence("chatRoomSeq")
		roomName = fmt.Sprintf("%d기", seq)
	} else {
		log.Printf("Create Couple Room, users: %v", matchEvent.MatchedUsers)
		startTime = time.Now()
		finishTime = startTime.Add(commontype.CoupleRunningTime)

		// TODO: 몇 기 게임방에서 커플 매칭이되었는지 추가
		seq = 0
		roomName = "커플 채팅방"
	}

	// 나는 솔로 캐릭터 할당
	male := 0
	female := 0

	for _, user := range matchEvent.MatchedUsers {
		var gamer models.GamerInfo

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

	room := models.ChatRoom{
		ID:                  chatRoomID,
		Name:                roomName,
		Seq:                 seq,
		Type:                matchEvent.MatchType,
		UserIDs:             extractUserIDs(matchEvent.MatchedUsers),
		Gamers:              gamers,
		Status:              commontype.RoomStatusGameStart,
		CreatedAt:           startTime,
		FinishChatAt:        finishTime,
		FinishFinalChoiceAt: finishTime.Add(commontype.FinishFinalChoiceTimer),
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

	// 밸런스 게임 타이머 설정 (15분)
	if matchEvent.MatchType == commontype.MATCH_GAME {
		err = s.redisClient.SetBalanceGameTimer(room.ID, commontype.BalanceGameStartTimer)
		if err != nil {
			log.Printf("Failed to set balance game timer: %v", err)
			return err
		}
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
func (s *ChatService) GetChatRoomList(userID int) ([]models.ChatRoom, error) {
	rooms, err := s.chatRepo.GetRoomsByUserID(userID)
	if err != nil {
		log.Printf("Failed to get chat rooms for user %d: %v", userID, err)
		return nil, err
	}
	return rooms, nil
}

func (s *ChatService) GetLatestMessage(roomID string) (*models.Chat, error) {
	return s.chatRepo.GetLastMessageByRoomID(roomID)
}

func (s *ChatService) GetUnreadCount(roomID string, userID int) (int, error) {
	return s.chatRepo.GetUnreadCountByUserAndRoom(userID, roomID)
}

func (s *ChatService) GetGamerInfo(userID int, roomID string) (*models.GamerInfo, error) {
	return s.chatRepo.GetUserGameInfoInRoom(userID, roomID)
}

func (s *ChatService) GetChatRoomByID(roomID string) (*models.ChatRoom, error) {
	return s.chatRepo.GetRoomByID(roomID)
}

// 특정 채팅방 상세 정보 조회
func (s *ChatService) GetChatRoomDetailByID(roomID string) (*dto.RoomDetailResponse, error) {
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
		user, err := s.userRepo.GetUserByID(userID)
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
			Address: commontype.Address(user.Address),
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
		reader := models.ChatReader{
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
func (s *ChatService) AddChatMsg(chatMsg models.Chat) (primitive.ObjectID, error) {
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
		reader := models.ChatReader{
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
func (s *ChatService) GetChatMsgListByRoomID(roomID string, pageNumber int, pageSize int) ([]*models.Chat, int64, error) {
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
func (s *ChatService) GetCharacterNameByRoomID(userID int, roomID string) (*models.GamerInfo, error) {
	return s.chatRepo.GetUserGameInfoInRoom(userID, roomID)
}

func (s *ChatService) IsUserInRoom(userID int, roomID string) (bool, error) {
	room, err := s.GetChatRoomByID(roomID)
	if err != nil {
		return false, err
	}
	if room == nil {
		return false, nil
	}

	// UserIDs 배열에 해당 사용자가 있는지 확인
	return lo.Contains(room.UserIDs, userID), nil
}

// 밸런스 게임 폼 삽입
func (s *ChatService) InsertBalanceForm(form *models.BalanceGameForm) (primitive.ObjectID, error) {
	return s.chatRepo.InsertBalanceForm(form)
}

// 밸런스 게임 폼 조회
func (s *ChatService) GetBalanceForm(formID primitive.ObjectID, userID int) (*dto.BalanceGameFormDTO, error) {
	form, err := s.chatRepo.GetBalanceFormByID(formID)
	if err != nil {
		return nil, err
	}

	myVote := commontype.BalanceFormVoteNone
	userVote, err := s.chatRepo.GetUserVote(formID, userID)
	if err != nil {
		return nil, err
	}
	if userVote != nil {
		myVote = userVote.Choiced
	}

	dto := dto.BalanceGameFormDTO{
		ID:       form.ID,
		RoomID:   form.RoomID,
		Question: form.Question,
		Votes:    form.Votes,
		Comments: form.Comments,
		MyVote:   myVote,
	}

	return &dto, nil
}

// 밸런스 게임 폼 투표 삽입
func (s *ChatService) InsertBalanceFormVote(vote *models.BalanceFormVote) error {
	return s.chatRepo.InsertBalanceFormVote(vote)
}

// 밸런스 게임 폼 투표 취소
func (s *ChatService) CancelBalanceFormVote(formID primitive.ObjectID, userID int) error {
	return s.chatRepo.CancelVote(formID, userID)
}

// 밸런스 게임 폼 댓글 삽입
func (s *ChatService) InsertBalanceFormComment(formID primitive.ObjectID, comment *models.BalanceFormComment) error {
	err := s.chatRepo.AddBalanceFormComment(formID, comment)
	if err != nil {
		log.Printf("Failed to insert balance form comment: %v", err)
		return err
	}

	roomID, err := s.chatRepo.GetRoomIdByBalanceFormID(formID)
	if err != nil {
		log.Printf("Failed to get room by balance form id: %v", err)
		return err
	}

	err = s.emitter.PublishVoteCommentChatEvent(eventtypes.VoteCommentChatEvent{
		FormID: formID,
		RoomID: roomID,
	})
	if err != nil {
		log.Printf("Failed to publish vote comment chat event: %v", err)
		return err
	}

	return nil
}

// 밸런스 게임 폼 댓글 조회
func (s *ChatService) GetBalanceFormComments(formID primitive.ObjectID, page int, pageSize int) ([]models.BalanceFormComment, int64, error) {
	return s.chatRepo.GetBalanceFormComments(formID, page, pageSize)
}

func (s *ChatService) SaveMatchHistory(matchHistory models.MatchHistory) {
	s.chatRepo.SaveMatchHistory(matchHistory)
}

func (s *ChatService) AddBalanceGameResult(roomSeq int, gameID primitive.ObjectID, winnerTeam int) error {
	result := models.BalanceGameResult{
		GameID:     gameID,
		WinnerTeam: winnerTeam,
	}

	err := s.chatRepo.UpdateMatchHistoryBalanceResult(roomSeq, result)
	if err != nil {
		log.Printf("Failed to add balance game result for room seq %d: %v", roomSeq, err)
		return err
	}

	return nil
}

func (s *ChatService) UpdateFinalMatch(roomSeq int, finalMatch []string) error {
	err := s.chatRepo.UpdateMatchHistoryFinalMatch(roomSeq, finalMatch)
	if err != nil {
		log.Printf("Failed to update final match for room seq %d: %v", roomSeq, err)
		return err
	}

	return nil
}

// GetBalanceFormsByRoomID returns all balance forms for a given room
func (s *ChatService) GetBalanceFormsByRoomID(roomID string) ([]models.BalanceGameForm, error) {
	return s.chatRepo.GetBalanceFormsByRoomID(roomID)
}

// DeleteBalanceFormVotes deletes all votes for a balance form
func (s *ChatService) DeleteBalanceFormVotes(formID primitive.ObjectID) error {
	return s.chatRepo.DeleteBalanceFormVotes(formID)
}

// DeleteBalanceFormComments deletes all comments for a balance form
func (s *ChatService) DeleteBalanceFormComments(formID primitive.ObjectID) error {
	return s.chatRepo.DeleteBalanceFormComments(formID)
}

// DeleteBalanceFormsByRoomID deletes all balance forms for a room
func (s *ChatService) DeleteBalanceFormsByRoomID(roomID string) error {
	return s.chatRepo.DeleteBalanceFormsByRoomID(roomID)
}

// DeleteMessageReaders deletes all message readers for a room
func (s *ChatService) DeleteMessageReaders(roomID string) error {
	return s.chatRepo.DeleteMessageReaders(roomID)
}
