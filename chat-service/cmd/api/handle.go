package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/baaami/dorandoran/chat/pkg/data"
	"github.com/baaami/dorandoran/chat/pkg/event"
	"github.com/baaami/dorandoran/chat/pkg/types"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// 방 생성
func (app *Config) createRoom(chatRoomCreateChan <-chan types.MatchEvent) {
	for matchEvent := range chatRoomCreateChan {
		// Create a unique ChatRoom ID (e.g., UUID or timestamp-based ID)
		chatRoomID := matchEvent.MatchId

		var seq int64
		var startTime time.Time
		var finishTime time.Time

		var gamers []data.GamerInfo

		if matchEvent.MatchType == types.MATCH_GAME {
			log.Printf("Create Game Room, users: %v", matchEvent.MatchedUsers)
			startTime = time.Now()
			// TODO: 시간 수정 필요
			finishTime = startTime.Add(5 * time.Minute)

			seq, _ = app.Models.ChatRoom.GetNextSequence("chatRoomSeq")
		} else {
			log.Printf("Create Couple Room, users: %v", matchEvent.MatchedUsers)
			startTime = time.Now()
			// TODO: 시간 수정 필요
			finishTime = startTime.Add(10 * time.Minute)

			seq = 0
		}

		// 나는 솔로 캐릭터 할당
		male := 0
		female := 0

		for _, user := range matchEvent.MatchedUsers {
			var gamer data.GamerInfo

			if matchEvent.MatchType == types.MATCH_GAME {
				gamer.UserID = user.ID
				if user.Gender == types.MALE {
					gamer.CharacterID = male
					gamer.CharacterName = data.MaleNames[male]
					male++
				} else {
					gamer.CharacterID = female
					gamer.CharacterName = data.FemaleNames[female]
					female++
				}

				gamer.CharacterAvatarURL = fmt.Sprintf("/profile?gender=%d&character_id=%d", user.Gender, gamer.CharacterID)
			} else {
				// 사용하지 않음
				gamer.CharacterID = -1
				gamer.CharacterAvatarURL = ""
			}

			gamers = append(gamers, gamer)
		}

		room := data.ChatRoom{
			ID:                  chatRoomID,
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
		err := app.Models.ChatRoom.InsertRoom(&room)
		if err != nil {
			log.Printf("Failed to insert chat room to MongoDB: %v", err)
			continue
		}

		ctx := context.Background()
		roomKey := fmt.Sprintf("room:%s", room.ID)

		// Redis에 채팅방 정보 생성
		err = app.RoomManager.RedisClient.Client.SAdd(ctx, roomKey, IntToStringArray(room.UserIDs)).Err()
		if err != nil {
			log.Printf("Failed to add room in redis, err: %s", err.Error())
			continue
		}

		app.RoomManager.RedisClient.SetRoomStatus(room.ID, types.RoomStatusGameIng)

		app.RoomManager.SetRoomTimeout(room.ID, time.Until(room.FinishChatAt))

		log.Printf("Chat room created: %s with users: %v", room.ID, room.UserIDs)

		if matchEvent.MatchType == types.MATCH_GAME {
			// 채팅방 생성 이벤트 발행
			err = app.Emitter.PublishChatRoomCreateEvent(room)
			if err != nil {
				log.Printf("Failed to publish chat room event: %v", err)
				continue
			}

			log.Printf("Published chat room event for room ID: %s", room.ID)
		} else {
			// 커플방 생성 이벤트 발행
			err = app.Emitter.PublishCoupleRoomCreateEvent(room)
			if err != nil {
				log.Printf("Failed to publish chat room event: %v", err)
				continue
			}

			log.Printf("Published chat room event for room ID: %s", room.ID)
		}
	}
}

// 채팅방 목록 조회
func (app *Config) getChatRoomList(w http.ResponseWriter, r *http.Request) {
	xUserID := r.Header.Get("X-User-ID")
	if xUserID == "" {
		http.Error(w, "User ID is required", http.StatusUnauthorized)
		log.Printf("User ID is required")
		return
	}

	userID, err := strconv.Atoi(xUserID)
	if err != nil {
		http.Error(w, fmt.Sprintf("User ID is not number, xUserID: %s", xUserID), http.StatusUnauthorized)
		log.Printf("User ID is not number, xUserID: %s", xUserID)
		return
	}

	// MongoDB에서 특정 유저가 참여한 채팅방 목록 조회
	rooms, err := app.Models.ChatRoom.GetRoomsByUserID(userID)
	if err != nil {
		http.Error(w, "Failed to retrieve chat rooms", http.StatusInternalServerError)
		return
	}

	// ChatRoomLatestResponse 배열을 생성
	var response []data.ChatRoomLatestResponse
	for _, room := range rooms {
		// 채팅방의 마지막 메시지 조회
		findLastMessage, err := app.Models.Chat.GetLastMessageByRoomID(room.ID)
		if err != nil {
			log.Printf("Failed to retrieve last message for room %s: %v", room.ID, err)
			continue
		}

		// 읽지 않은 메시지 개수 조회
		unreadCount, err := app.Models.ChatReader.GetUnreadCountByUserAndRoom(userID, room.ID)
		if err != nil {
			log.Printf("Failed to retrieve unread count for user %d in room %s: %v", userID, room.ID, err)
			unreadCount = 0
		}

		gamerInfo, err := app.Models.ChatRoom.GetUserGameInfoInRoom(findLastMessage.SenderID, room.ID)
		if err != nil {
			if err.Error() == "user not found in the game" {
				log.Printf("user not found in the game")
			} else {
				log.Printf("Failed to GetUserGameInfoInRoom, user %d in room %s, err: %v", findLastMessage.SenderID, room.ID, err)
				continue
			}
		}
		if gamerInfo == nil {
			log.Printf("gamerinfo is nil")
			continue
		}

		lastMessage := data.LastMessage{
			SenderID: findLastMessage.SenderID,
			Message:  findLastMessage.Message,
			GameInfo: types.GameInfo{
				CharacterID:        gamerInfo.CharacterID,
				CharacterName:      gamerInfo.CharacterName,
				CharacterAvatarURL: gamerInfo.CharacterAvatarURL,
			},
			CreatedAt: findLastMessage.CreatedAt,
		}

		roomName := getRoomName(room, userID)

		chatRoomResponse := data.ChatRoomLatestResponse{
			ID:          room.ID,
			RoomName:    roomName,  // 필요시 동적으로 추가
			RoomType:    room.Type, // 0: 게임방, 1: 커플방
			LastMessage: lastMessage,
			UnreadCount: unreadCount,
			CreatedAt:   room.CreatedAt,
			ModifiedAt:  room.ModifiedAt,
		}

		// 배열에 추가
		response = append(response, chatRoomResponse)
	}

	// 결과 반환
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	w.WriteHeader(http.StatusOK)
}

func getRoomName(room data.ChatRoom, userID int) string {
	var roomName string

	if room.Type == types.MATCH_GAME {
		// 게임방의 경우 기수를 이름에 할당
		roomName = fmt.Sprintf("%d기", room.Seq)
	} else {
		// 커플방의 경우 상대방 이름을 할당
		var partnerID int

		// 자신의 ID가 아닌 다른 사용자 ID를 partnerID에 할당
		for _, id := range room.UserIDs {
			if id != userID {
				partnerID = id
				break
			}
		}

		user, err := getUserByUserID(strconv.Itoa(partnerID))
		if err != nil {
			log.Printf("Failed to getUserByUserID, userID: %d, err: %s", partnerID, err.Error())
			roomName = fmt.Sprintf("%d기 커플방", room.Seq)
			return roomName
		}

		// 상대방 ID로 원하는 작업 수행 (예: 이름 가져오기)
		roomName = user.Name
	}

	return roomName
}

// 채팅방 상세 정보 조회
func (app *Config) getChatRoomByID(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "id")

	// MongoDB에서 Room ID로 채팅방 조회
	room, err := app.Models.ChatRoom.GetRoomByID(roomID)
	if err != nil {
		http.Error(w, "Failed to find chat room", http.StatusInternalServerError)
		return
	}
	if room == nil {
		http.Error(w, "Chat room not found", http.StatusNotFound)
		return
	}

	var gamerList []types.Gamer

	for _, userID := range room.UserIDs {
		user, err := getUserByUserID(strconv.Itoa(userID))
		if err != nil {
			log.Printf("Failed to get user, id: %d, err: %s", userID, err.Error())
			continue
		}
		if user == nil {
			log.Printf("Cannot find user in room, user id: %d, room id: %s", userID, roomID)
			continue
		}

		// room 내 해당 user의 정보
		gamerInfo, err := app.Models.ChatRoom.GetUserGameInfoInRoom(userID, roomID)
		if err != nil {
			if err.Error() == "user not found in the game" {
				log.Printf("user not found in the game")
			} else {
				log.Printf("Failed to GetUserGameInfoInRoom, user %d in room %s, err: %v", user.ID, room.ID, err)
				continue
			}
		}

		gamer := types.Gamer{
			ID:      user.ID,
			SnsType: user.SnsType,
			SnsID:   user.SnsID,
			Name:    user.Name,
			Gender:  user.Gender,
			Birth:   user.Birth,
			Address: user.Address,
			GameInfo: types.GameInfo{
				CharacterID:        gamerInfo.CharacterID,
				CharacterName:      gamerInfo.CharacterName,
				CharacterAvatarURL: gamerInfo.CharacterAvatarURL,
			},
		}

		gamerList = append(gamerList, gamer)
	}

	payload := data.RoomDetailResponse{
		ID:                  room.ID,
		Type:                room.Type,
		Status:              room.Status,
		Users:               gamerList,
		CreatedAt:           room.CreatedAt,
		FinishChatAt:        room.FinishChatAt,
		FinishFinalChoiceAt: room.FinishFinalChoiceAt,
		ModifiedAt:          room.ModifiedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

// 특정 방의 채팅 목록 조회
func (app *Config) getChatMsgListByRoomID(w http.ResponseWriter, r *http.Request) {
	// URL에서 room ID 가져오기
	roomID := chi.URLParam(r, "id")
	if roomID == "" {
		http.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}

	// 쿼리 매개변수로 페이지 번호와 페이지 크기 가져오기
	page := r.URL.Query().Get("page")
	pageNumber := 1
	if page != "" {
		if parsedPage, err := strconv.Atoi(page); err == nil {
			pageNumber = parsedPage
		}
	}
	// MongoDB에서 데이터 가져오기
	messages, totalCount, err := app.Models.Chat.GetByRoomIDWithPagination(roomID, pageNumber, data.PAGE_DEFAULT_SIZE)
	if err != nil {
		log.Printf("Failed to GetByRoomIDWithPagination, err: %v", err)
		http.Error(w, "Failed to fetch chat messages", http.StatusInternalServerError)
		return
	}

	// 총 페이지 수 및 hasNextPage 계산
	totalPages := int((totalCount + int64(data.PAGE_DEFAULT_SIZE) - 1) / int64(data.PAGE_DEFAULT_SIZE)) // 올림 계산
	hasNextPage := pageNumber < totalPages

	// 응답 생성
	response := data.ChatListResponse{
		Data:        messages,
		CurrentPage: pageNumber,
		NextPage:    pageNumber + 1,
		HasNextPage: hasNextPage,
		TotalPages:  totalPages,
	}

	// JSON 응답 전송
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode chat messages", http.StatusInternalServerError)
		return
	}
}

// 게임방 내 캐릭터명 조회
func (app *Config) getCharacterNameByRoomID(w http.ResponseWriter, r *http.Request) {
	xUserID := r.Header.Get("X-User-ID")
	if xUserID == "" {
		http.Error(w, "User ID is required", http.StatusUnauthorized)
		log.Printf("User ID is required")
		return
	}

	userID, err := strconv.Atoi(xUserID)
	if err != nil {
		http.Error(w, fmt.Sprintf("User ID is not number, xUserID: %s", xUserID), http.StatusUnauthorized)
		log.Printf("User ID is not number, xUserID: %s", xUserID)
		return
	}

	// URL에서 room ID 가져오기
	roomID := chi.URLParam(r, "id")
	if roomID == "" {
		http.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}

	gamerInfo, err := app.Models.ChatRoom.GetUserGameInfoInRoom(userID, roomID)
	if err != nil {
		if err.Error() == "user not found in the game" {
			log.Printf("user not found in the game")
		} else {
			log.Printf("Failed to GetUserGameInfoInRoom, user %d in room %s, err: %v", userID, roomID, err)
			http.Error(w, "failed to GetUserGameInfoInRoom", http.StatusInternalServerError)
			return
		}
	}

	// JSON 응답 전송
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(*gamerInfo); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode chat messages", http.StatusInternalServerError)
		return
	}
}

// 특정 Room ID로 채팅방 삭제
func (app *Config) deleteChatRoom(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "id")

	// room, err := app.Models.ChatRoom.GetRoomByID(roomID)
	// if err != nil {
	// 	log.Printf("Failed to get chat room, id: %s, err: %s", roomID, err.Error())
	// 	http.Error(w, "Failed to get chat room", http.StatusInternalServerError)
	// 	return
	// }

	// MongoDB에서 Room ID로 채팅방 삭제
	err := app.Models.ChatRoom.DeleteRoom(roomID)
	if err != nil {
		http.Error(w, "Failed to delete chat room", http.StatusInternalServerError)
		return
	}

	ctx := context.Background()
	roomKey := fmt.Sprintf("room:%s", roomID)

	// Redis에 채팅방 정보 삭제
	err = app.RoomManager.RedisClient.Client.Del(ctx, roomKey).Err()
	if err != nil {
		http.Error(w, "Failed to delete room in redis", http.StatusInternalServerError)
		log.Printf("Failed to delete room in redis, err: %s", err.Error())
		return
	}

	// TODO: 채팅방 삭제 이벤트 발행

	w.WriteHeader(http.StatusOK)
	log.Printf("Chat room deleted, roomID: %s", roomID)
}

// 채팅방 나가기
func (app *Config) leaveChatRoom(w http.ResponseWriter, r *http.Request) {
	xUserID := r.Header.Get("X-User-ID")
	if xUserID == "" {
		http.Error(w, "User ID is required", http.StatusUnauthorized)
		log.Printf("User ID is required")
		return
	}

	userID, err := strconv.Atoi(xUserID)
	if err != nil {
		http.Error(w, fmt.Sprintf("User ID is not number, xUserID: %s", xUserID), http.StatusUnauthorized)
		log.Printf("User ID is not number, xUserID: %s", xUserID)
		return
	}

	roomID := chi.URLParam(r, "id")

	err = app.Models.ChatRoom.LeaveRoom(roomID, userID)
	if err != nil {
		http.Error(w, "Failed to leave chat room", http.StatusInternalServerError)
		return
	}

	ctx := context.Background()
	roomKey := fmt.Sprintf("room:%s", roomID)

	// Redis에서 유저 제거
	err = app.RoomManager.RedisClient.Client.SRem(ctx, roomKey, strconv.Itoa(userID)).Err()
	if err != nil {
		log.Printf("Failed to remove user %d from Redis room %s: %v", userID, roomKey, err)
		http.Error(w, "Failed to update Redis room", http.StatusInternalServerError)
		return
	}

	// 채팅방 나가기 이벤트 발행
	err = app.Emitter.PushRoomLeaveEvent(event.RoomLeaveEvent{
		LeaveUserID: userID,
		RoomID:      roomID,
	})
	if err != nil {
		log.Printf("Failed to PushRoomLeaveEvent user %d from room %s, err: %v", userID, roomID, err)
		http.Error(w, "Failed to PushRoomLeaveEvent", http.StatusInternalServerError)
		return
	}

	// TODO: 채팅방 삭제 이벤트 발행

	w.WriteHeader(http.StatusOK)
	log.Printf("Chat room deleted, roomID: %s", roomID)
}

// 채팅 메시지 추가
func (app *Config) addChatMsg(w http.ResponseWriter, r *http.Request) {
	var chatEventMsg data.ChatEvent
	err := json.NewDecoder(r.Body).Decode(&chatEventMsg)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	log.Printf("ADD chatMsg: %v", chatEventMsg)

	// Chat에 삽입
	chat := data.Chat{
		MessageId:   chatEventMsg.MessageId,
		Type:        chatEventMsg.Type,
		RoomID:      chatEventMsg.RoomID,
		SenderID:    chatEventMsg.SenderID,
		Message:     chatEventMsg.Message,
		UnreadCount: chatEventMsg.UnreadCount,
		CreatedAt:   chatEventMsg.CreatedAt,
	}

	// Insert the chat and get the generated _id
	messageID, err := app.Models.Chat.Insert(chat)
	if err != nil {
		log.Printf("Failed to insert chat message: %v", err)
		http.Error(w, "Failed to insert chat message", http.StatusInternalServerError)
		return
	}

	log.Printf("Chat message inserted with ID: %s", messageID.Hex())

	if len(chatEventMsg.ReaderIds) > 0 {
		err = app.addChatReaders(messageID, chat.RoomID, chatEventMsg.ReaderIds, chat.CreatedAt)
		if err != nil {
			log.Printf("Failed to process chat readers: %v", err)
			http.Error(w, "Failed to process chat readers", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Chat message inserted successfully"))
	log.Printf("Chat message from %d in room[%s]", chat.SenderID, chat.RoomID)
}

func (app *Config) addChatReaders(messageID primitive.ObjectID, roomID string, readerIDs []int, readAt time.Time) error {
	for _, userID := range readerIDs {
		// ChatReader 데이터 생성
		reader := data.ChatReader{
			MessageId: messageID,
			RoomID:    roomID,
			UserId:    userID,
			ReadAt:    readAt,
		}

		// 데이터베이스에 삽입
		err := app.Models.ChatReader.Insert(reader)
		if err != nil {
			log.Printf("Failed to insert ChatReader for user %d: %v", userID, err)
			return fmt.Errorf("failed to insert ChatReader: %w", err)
		}
	}

	log.Printf("Successfully processed chat.read event for MessageId: %s", messageID.Hex())
	return nil
}

// 채팅 메시지 읽음 처리
func (app *Config) handleChatRead(w http.ResponseWriter, r *http.Request) {
	// 요청 바디 파싱
	var readersEvent data.ChatReadersEvent
	err := json.NewDecoder(r.Body).Decode(&readersEvent)
	if err != nil {
		log.Printf("Failed to decode request payload: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	log.Printf("Received ChatReadersEvent: %+v", readersEvent)

	// 각 사용자에 대해 ChatReader를 생성 및 삽입
	for _, userID := range readersEvent.UserIds {
		userIDInt, err := strconv.Atoi(userID)
		if err != nil {
			log.Printf("Invalid user ID %s: %v", userID, err)
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		// ChatReader 데이터 생성
		reader := data.ChatReader{
			MessageId: readersEvent.MessageId,
			RoomID:    readersEvent.RoomID,
			UserId:    userIDInt,
			ReadAt:    readersEvent.ReadAt,
		}

		// 데이터베이스에 삽입
		err = app.Models.ChatReader.Insert(reader)
		if err != nil {
			log.Printf("Failed to insert ChatReader for user %d: %v", userIDInt, err)
			http.Error(w, "Failed to process chat.read event", http.StatusInternalServerError)
			return
		}
	}

	// 성공 응답
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Chat read event processed successfully"))
	log.Printf("Successfully processed chat.read event for MessageId: %s", readersEvent.MessageId.Hex())
}

// handleRoomJoin processes a room.join event and inserts read data for messages created before JoinAt
func (app *Config) handleRoomJoin(w http.ResponseWriter, r *http.Request) {
	// 요청 바디 파싱
	var roomJoinEvent data.RoomJoinEvent
	err := json.NewDecoder(r.Body).Decode(&roomJoinEvent)
	if err != nil {
		log.Printf("Failed to decode room join event: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	log.Printf("Processing RoomJoinEvent: %+v", roomJoinEvent)

	// TODO: 메시지를 가져오지 않고 바로 작업을 할 수 있지 않을까??
	// 읽지 않은 JoinAt 이전의 메시지 가져오기
	messages, err := app.Models.Chat.GetUnreadMessagesBefore(roomJoinEvent.RoomID, roomJoinEvent.JoinAt, roomJoinEvent.UserID)
	if err != nil {
		log.Printf("Failed to get messages for RoomID %s: %v", roomJoinEvent.RoomID, err)
		http.Error(w, "Failed to get messages", http.StatusInternalServerError)
		return
	}

	// 읽음 처리
	var messageIDs []primitive.ObjectID
	for _, message := range messages {
		// 읽음 처리
		reader := data.ChatReader{
			MessageId: message.MessageId,
			RoomID:    roomJoinEvent.RoomID,
			UserId:    roomJoinEvent.UserID,
			ReadAt:    roomJoinEvent.JoinAt,
		}

		err := app.Models.ChatReader.Insert(reader)
		if err != nil {
			log.Printf("Failed to insert ChatReader for MessageId %s: %v", message.MessageId.Hex(), err)
			continue
		}

		messageIDs = append(messageIDs, message.MessageId)
	}

	if len(messageIDs) > 0 {
		// unread_count 업데이트
		err = app.Models.Chat.UpdateUnreadCounts(messageIDs)
		if err != nil {
			log.Printf("Failed to update unread counts for RoomID %s: %v", roomJoinEvent.RoomID, err)
			http.Error(w, "Failed to update unread counts", http.StatusInternalServerError)
			return
		}

		app.Emitter.PushChatLatestEvent(event.ChatLatestEvent{
			RoomID: roomJoinEvent.RoomID,
		})

		log.Printf("Successfully processed RoomJoinEvent for RoomID: %s, UserID: %d", roomJoinEvent.RoomID, roomJoinEvent.UserID)
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Room join event processed successfully"))
}

// 채팅 메시지 삭제 (by ChatRoom)
func (app *Config) deleteChatByRoomID(w http.ResponseWriter, r *http.Request) {
	// URL에서 room ID 가져오기
	roomID := chi.URLParam(r, "id")

	err := app.Models.Chat.DeleteChatByRoomID(roomID)
	if err != nil {
		log.Printf("Failed to Delete Chat Data, roomID: %s, err: %s", roomID, err)
		http.Error(w, "Failed to Delete Chat Data, roomID", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}

// [Bridge user] 회원 정보 획득
func getUserByUserID(userID string) (*types.User, error) {
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
	var user types.User

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

func extractUserIDs(users []types.WaitingUser) []int {
	ids := make([]int, len(users))
	for i, user := range users {
		ids[i] = user.ID
	}
	return ids
}
