package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type User struct {
	ID       int    `json:"id"`
	SnsType  string `json:"sns_type"`
	SnsID    string `json:"sns_id"`
	Name     string `json:"name"`
	Nickname string `json:"nickname"`
	Gender   int    `json:"gender"`
	Age      int    `json:"age"`
	Email    string `json:"email"`
}

// 유저 정보 조회
func (app *Config) readUser(w http.ResponseWriter, r *http.Request) {
	// URL에서 유저 ID 가져오기
	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// DB에서 유저 정보 조회
	user, err := app.Models.GetUserByID(userID)
	if err != nil {
		http.Error(w, "Failed to retrieve user", http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	log.Printf("Select User: %v", user)

	// JSON으로 응답 반환
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// 유저 정보 삽입
func (app *Config) insertUser(w http.ResponseWriter, r *http.Request) {
	var newUser User

	// 요청에서 유저 데이터를 읽음
	err := json.NewDecoder(r.Body).Decode(&newUser)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// DB에 유저 삽입
	insertedID, err := app.Models.InsertUser(newUser.Name, newUser.Nickname, newUser.Gender, newUser.Age, newUser.Email)
	if err != nil {
		http.Error(w, "Failed to insert user", http.StatusInternalServerError)
		return
	}

	log.Printf("Insert User: %v", newUser)

	// 삽입된 유저 정보 반환
	newUser.ID = int(insertedID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newUser)
}

// 유저 정보 업데이트
func (app *Config) updateUser(w http.ResponseWriter, r *http.Request) {
	// URL에서 유저 ID 가져오기
	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var updatedUser User

	// 요청에서 유저 데이터를 읽음
	err = json.NewDecoder(r.Body).Decode(&updatedUser)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	updatedUser.ID = userID

	// DB에서 유저 정보 업데이트
	err = app.Models.UpdateUser(updatedUser.ID, updatedUser.Name, updatedUser.Nickname, updatedUser.Gender, updatedUser.Age, updatedUser.Email)
	if err != nil {
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	log.Printf("Update User: %v", updatedUser)

	// 업데이트된 유저 정보 반환
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedUser)
}

// 유저 정보 삭제
func (app *Config) deleteUser(w http.ResponseWriter, r *http.Request) {
	// URL에서 유저 ID 가져오기
	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// DB에서 유저 삭제
	err = app.Models.DeleteUser(userID)
	if err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	log.Printf("Delete User: %v", userID)

	// 성공 메시지 반환
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User deleted"})
}
