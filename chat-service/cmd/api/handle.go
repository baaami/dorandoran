package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

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
