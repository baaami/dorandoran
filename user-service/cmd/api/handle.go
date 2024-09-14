package main

import (
	"encoding/json"
	"net/http"
)

// 가상의 유저 데이터를 저장하는 간단한 구조체
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var users = []User{
	{ID: 1, Name: "John Doe", Email: "john@example.com"},
	{ID: 2, Name: "Chulsu", Email: "chulsu@example.com"},
}

// 유저 정보 조회
func (app *Config) readUser(w http.ResponseWriter, r *http.Request) {
	// 전체 유저 정보를 반환
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
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

	// 새로운 유저 추가
	newUser.ID = len(users) + 1
	users = append(users, newUser)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newUser)
}

// 유저 정보 업데이트
func (app *Config) updateUser(w http.ResponseWriter, r *http.Request) {
	var updatedUser User

	// 요청에서 유저 데이터를 읽음
	err := json.NewDecoder(r.Body).Decode(&updatedUser)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// 유저 정보를 업데이트
	for i, user := range users {
		if user.ID == updatedUser.ID {
			users[i] = updatedUser
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(updatedUser)
			return
		}
	}

	http.Error(w, "User not found", http.StatusNotFound)
}

// 유저 정보 삭제
func (app *Config) deleteUser(w http.ResponseWriter, r *http.Request) {
	var deletedUser User

	// 요청에서 유저 데이터를 읽음
	err := json.NewDecoder(r.Body).Decode(&deletedUser)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// 유저 정보를 삭제
	for i, user := range users {
		if user.ID == deletedUser.ID {
			users = append(users[:i], users[i+1:]...)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "User deleted"})
			return
		}
	}

	http.Error(w, "User not found", http.StatusNotFound)
}
