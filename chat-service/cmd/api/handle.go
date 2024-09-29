package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/baaami/dorandoran/chat/cmd/data"
	"github.com/baaami/dorandoran/chat/pkg/event"
	"github.com/go-chi/chi/v5"
)

type Chat struct {
	RoomID    string    `bson:"room_id" json:"room_id"`
	SenderID  string    `bson:"sender_id" json:"sender_id"`
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
func (app *Config) getChatRoomsByUserID(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")

	// MongoDB에서 특정 유저가 참여한 채팅방 목록 조회
	rooms, err := app.Models.ChatRoom.GetRoomsByUserID(userID)
	if err != nil {
		http.Error(w, "Failed to retrieve chat rooms", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rooms)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room)
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
	json.NewEncoder(w).Encode(map[string]string{"message": "Chat room deleted"})
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
		RoomID:   chatMsg.RoomID,
		SenderID: chatMsg.SenderID,
		Message:  chatMsg.Message,
	}

	err = app.Models.Chat.Insert(entry)
	if err != nil {
		http.Error(w, "Failed to insert chat message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Chat message inserted successfully"))
	log.Printf("Chat message from %s in room[%s]", chatMsg.SenderID, chatMsg.RoomID)
}

// 특정 방의 채팅 메시지 리스트 획득
func (app *Config) getChatMsgListByRoomID(w http.ResponseWriter, r *http.Request) {
	// URL에서 room ID 가져오기
	roomID := chi.URLParam(r, "id")

	messages, err := app.Models.Chat.GetByRoomID(roomID)
	if err != nil {
		log.Printf("Failed to GetByRoomID, err: %v", err)
		http.Error(w, "Failed to Chat", http.StatusInternalServerError)
		return
	}

	// 결과를 JSON으로 변환하여 반환
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(messages); err != nil {
		log.Printf("Failed to encode messages, msg: %v, err: %v", messages, err)
		http.Error(w, "Failed to encode chat messages", http.StatusInternalServerError)
		return
	}
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
