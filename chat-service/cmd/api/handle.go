package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/baaami/dorandoran/chat/pkg/data"
	"github.com/baaami/dorandoran/chat/pkg/event"
	common "github.com/baaami/dorandoran/common/user"
	"github.com/go-chi/chi/v5"
	"github.com/samber/lo"
)

type Chat struct {
	Type      string    `bson:"type" json:"type"`
	RoomID    string    `bson:"room_id" json:"room_id"`
	SenderID  int       `bson:"sender_id" json:"sender_id"`
	Message   string    `bson:"message" json:"message"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

// 채팅방 생성
func (app *Config) createChatRoom(w http.ResponseWriter, r *http.Request) {
	var room data.ChatRoom

	// 요청 바디에서 데이터 읽기
	err := json.NewDecoder(r.Body).Decode(&room)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// MongoDB에 새로운 채팅방 삽입
	err = app.Models.ChatRoom.InsertRoom(&room)
	if err != nil {
		http.Error(w, "Failed to create chat room", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(room)
}

// 특정 유저의 채팅방 목록 조회
func (app *Config) getChatRoomList(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusUnauthorized)
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

		// 사용자의 마지막 읽은 시간 가져오기
		lastReadTime, ok := room.UserLastRead[userID]
		if !ok {
			lastReadTime = time.Time{} // Default to zero time if no last read time is found
		}

		// 읽지 않은 메시지 수 계산
		unreadCount, err := app.Models.Chat.GetUnreadMessageCount(room.ID, userID, lastReadTime)
		if err != nil {
			log.Printf("Failed to calculate unread count for room %s: %v", room.ID, err)
			unreadCount = 0
		}

		lastMessage := data.LastMessage{
			SenderID:  findLastMessage.SenderID,
			Message:   findLastMessage.Message,
			CreatedAt: findLastMessage.CreatedAt,
		}

		// ChatRoomLatestResponse 생성
		chatRoomResponse := data.ChatRoomLatestResponse{
			ID:          room.ID,
			RoomName:    "채팅방 이름", // 필요시 동적으로 추가
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

// Room ID로 채팅방 상세 정보 조회
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

	var userList []common.User

	for _, userID := range room.Users {
		user, err := getUserByUserID(userID)
		if err != nil {
			log.Printf("Failed to get user, id: %s, err: %s", userID, err.Error())
			continue
		}
		if user == nil {
			log.Printf("Cannot find user in room, id: %s, room id: %s", userID, roomID)
			continue
		}

		userList = append(userList, *user)
	}

	payload := data.ChatRoomDetailResponse{
		ID:           room.ID,
		Users:        userList,
		CreatedAt:    room.CreatedAt,
		ModifiedAt:   room.ModifiedAt,
		UserLastRead: room.UserLastRead,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

// 특정 방의 채팅 메시지 리스트 획득
func (app *Config) getChatMsgListByRoomID(w http.ResponseWriter, r *http.Request) {
	// URL에서 room ID 가져오기
	roomID := chi.URLParam(r, "id")

	// 쿼리 매개변수로 페이지 번호와 페이지 크기 가져오기
	page := r.URL.Query().Get("page")

	// 기본값 설정: 페이지 번호는 1, limit는 50으로 설정
	pageNumber := 1
	pageSize := 50

	if page != "" {
		pageNumber, _ = strconv.Atoi(page)
	}

	messages, err := app.Models.Chat.GetByRoomIDWithPagination(roomID, pageNumber, pageSize)
	if err != nil {
		log.Printf("Failed to GetByRoomIDWithPagination, err: %v", err)
		http.Error(w, "Failed to Chat", http.StatusInternalServerError)
		return
	}

	if messages == nil {
		messages = []*data.Chat{}
	}

	// 결과를 JSON으로 변환하여 반환
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(messages); err != nil {
		log.Printf("Failed to encode messages, msg: %v, err: %v", messages, err)
		http.Error(w, "Failed to encode chat messages", http.StatusInternalServerError)
		return
	}
}

func (app *Config) confirmChatRoom(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "room_id")
	userID := chi.URLParam(r, "user_id")

	// 채팅방이 존재하는지 확인
	room, err := app.Models.ChatRoom.GetRoomByID(roomID)
	if err != nil {
		log.Printf("Failed to find chat room, roomID: %s", roomID)
		http.Error(w, "Failed to find chat room", http.StatusInternalServerError)
		return
	}
	if room == nil {
		log.Printf("Failed to find chat room, roomID: %s", roomID)
		http.Error(w, "Chat room not found", http.StatusNotFound)
		return
	}

	if !lo.Contains(room.Users, userID) {
		log.Printf("User is not a member of the chat room, roomID: %s, userID: %s", roomID, userID)
		http.Error(w, "User is not a member of the chat room", http.StatusForbidden)
		return
	}

	// 채팅방의 UserLastRead 필드를 업데이트
	err = app.Models.ChatRoom.ConfirmRoom(roomID, userID)
	if err != nil {
		http.Error(w, "Failed to confirm chat room", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Printf("Chat room confirmed, roomID: %s, userID: %s", roomID, userID)
}

func (app *Config) confirmChatRoomByUser(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "room_id")

	// 사용자 ID를 가져옵니다. (예: 헤더에서 "X-User-ID"로 전달된다고 가정)
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusUnauthorized)
		return
	}

	// 채팅방이 존재하는지 확인
	room, err := app.Models.ChatRoom.GetRoomByID(roomID)
	if err != nil {
		log.Printf("Failed to find chat room, roomID: %s", roomID)
		http.Error(w, "Failed to find chat room", http.StatusInternalServerError)
		return
	}
	if room == nil {
		log.Printf("Failed to find chat room, roomID: %s", roomID)
		http.Error(w, "Chat room not found", http.StatusNotFound)
		return
	}

	// 사용자가 해당 채팅방의 멤버인지 확인
	if !lo.Contains(room.Users, userID) {
		log.Printf("User is not a member of the chat room, roomID: %s, userID: %s", roomID, userID)
		http.Error(w, "User is not a member of the chat room", http.StatusForbidden)
		return
	}

	// 채팅방의 UserLastRead 필드를 업데이트
	err = app.Models.ChatRoom.ConfirmRoom(roomID, userID)
	if err != nil {
		http.Error(w, "Failed to confirm chat room", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Printf("Chat room confirmed, roomID: %s, userID: %s", roomID, userID)
}

// 특정 Room ID로 채팅방 삭제
func (app *Config) deleteChatRoom(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "id")

	room, err := app.Models.ChatRoom.GetRoomByID(roomID)
	if err != nil {
		log.Printf("Failed to get chat room, id: %s, err: %s", roomID, err.Error())
		http.Error(w, "Failed to get chat room", http.StatusInternalServerError)
		return
	}

	// MongoDB에서 Room ID로 채팅방 삭제
	err = app.Models.ChatRoom.DeleteRoom(roomID)
	if err != nil {
		http.Error(w, "Failed to delete chat room", http.StatusInternalServerError)
		return
	}

	// 채팅방 삭제 이벤트 발행
	emitter, err := event.NewEventEmitter(app.Rabbit)
	if err == nil {
		log.Printf("[INFO] Pushing Room Delete Event to RabbitMQ, room: %s", roomID)
		emitter.PushRoomToQueue(*room)
	} else {
		log.Printf("[ERROR] Failed to create event emitter: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	log.Printf("Chat room deleted, roomID: %s", roomID)
}

// 채팅 메시지 추가
func (app *Config) addChatMsg(w http.ResponseWriter, r *http.Request) {
	var chatMsg Chat
	err := json.NewDecoder(r.Body).Decode(&chatMsg)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	log.Printf("ADD chatMsg: %v", chatMsg)

	// Chat에 삽입
	entry := data.Chat{
		Type:      chatMsg.Type,
		RoomID:    chatMsg.RoomID,
		SenderID:  chatMsg.SenderID,
		Message:   chatMsg.Message,
		CreatedAt: chatMsg.CreatedAt,
	}

	err = app.Models.Chat.Insert(entry)
	if err != nil {
		http.Error(w, "Failed to insert chat message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Chat message inserted successfully"))
	log.Printf("Chat message from %d in room[%s]", chatMsg.SenderID, chatMsg.RoomID)
}

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
func getUserByUserID(userID string) (*common.User, error) {
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
	var user common.User

	// 응답 본문 로깅 추가
	body, err := ioutil.ReadAll(resp.Body)
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
