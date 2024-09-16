package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/baaami/dorandoran/chat/cmd/data"
	"github.com/go-chi/chi/v5"
)

// ChatMessage 구조체 정의
type ChatMessage struct {
	RoomID     string    `bson:"room_id"`
	SenderID   string    `bson:"sender_id"`
	ReceiverID string    `bson:"receiver_id"`
	Message    string    `bson:"message"`
	CreatedAt  time.Time `bson:"created_at"`
}

// 채팅방 구조체
type ChatRoom struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// 채팅방 목록 (메모리 상에서 관리)
var chatRooms = []ChatRoom{}
var nextID = 1

// 채팅방 생성 API
func (app *Config) createChatRoom(w http.ResponseWriter, r *http.Request) {
	var newChatRoom ChatRoom

	// 요청에서 채팅방 정보를 읽음
	err := json.NewDecoder(r.Body).Decode(&newChatRoom)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// ID 자동 증가
	newChatRoom.ID = nextID
	nextID++

	// 채팅방을 목록에 추가
	chatRooms = append(chatRooms, newChatRoom)

	// 생성된 채팅방 정보를 반환
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newChatRoom)
}

// 채팅방 목록 불러오기 API
func (app *Config) getChatRooms(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatRooms)
}

// 채팅방 삭제 API
func (app *Config) deleteChatRoom(w http.ResponseWriter, r *http.Request) {
	// URL 경로에서 채팅방 ID를 가져옴
	idParam := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		http.Error(w, "Invalid chat room ID", http.StatusBadRequest)
		return
	}

	// 채팅방을 찾고 삭제
	for i, room := range chatRooms {
		if room.ID == id {
			chatRooms = append(chatRooms[:i], chatRooms[i+1:]...)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message": "Chat room deleted"}`))
			return
		}
	}

	// 채팅방을 찾지 못한 경우
	http.Error(w, "Chat room not found", http.StatusNotFound)
}

// 채팅 메시지를 추가하는 핸들러
func (app *Config) addChatMsg(w http.ResponseWriter, r *http.Request) {
	var chatMsg ChatMessage
	err := json.NewDecoder(r.Body).Decode(&chatMsg)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// ChatEntry에 삽입
	entry := data.ChatEntry{
		RoomID:     chatMsg.RoomID,
		SenderID:   chatMsg.SenderID,
		ReceiverID: chatMsg.ReceiverID,
		Message:    chatMsg.Message,
	}

	err = app.Models.ChatEntry.Insert(entry)
	if err != nil {
		http.Error(w, "Failed to insert chat message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Chat message inserted successfully"))
	log.Printf("Chat message from %s to %s inserted", chatMsg.SenderID, chatMsg.ReceiverID)
}

func (app *Config) getChatMsgList(w http.ResponseWriter, r *http.Request) {
	// URL에서 room ID 가져오기
	roomID := chi.URLParam(r, "id")

	messages, err := app.Models.ChatEntry.GetByRoomID(roomID)
	if err != nil {
		log.Printf("Failed to GetByRoomID, err: %v", err)
		http.Error(w, "Failed to chatentry", http.StatusInternalServerError)
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
