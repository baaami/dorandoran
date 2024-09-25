package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

type User struct {
	ID       int    `json:"id"`
	SnsType  int    `json:"sns_type"`
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
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		log.Printf("Failed to Atoi user ID, err: %s", err.Error())
		http.Error(w, "Failed to Atoi user ID", http.StatusInternalServerError)
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

// 유저 정보 조회
func (app *Config) checkUserExistence(w http.ResponseWriter, r *http.Request) {
	// 쿼리 파라미터에서 sns_type과 sns_id를 가져옴
	snsType := r.URL.Query().Get("sns_type")
	snsID := r.URL.Query().Get("sns_id")

	// sns_type이나 sns_id가 없는 경우 오류 반환
	if snsType == "" || snsID == "" {
		http.Error(w, "Missing sns_type or sns_id", http.StatusBadRequest)
		return
	}

	nSnsType, _ := strconv.Atoi(snsType)

	// DB에서 사용자 조회
	user, err := app.Models.GetUserBySNS(nSnsType, snsID)
	if err != nil {
		http.Error(w, "Error fetching user", http.StatusInternalServerError)
		return
	}

	// 유저가 존재하지 않는 경우
	if user == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// 유저가 존재하는 경우, StatusOK와 함께 유저 정보 반환
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

// 유저 정보 삽입
func (app *Config) registerUser(w http.ResponseWriter, r *http.Request) {
	var newUser User

	// 요청에서 sns_type과 sns_id만 읽음
	err := json.NewDecoder(r.Body).Decode(&newUser)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// 나머지 필드들을 기본값으로 초기화
	newUser.Name = ""     // 빈 문자열로 초기화
	newUser.Nickname = "" // 빈 문자열로 초기화
	newUser.Gender = 0    // 0으로 초기화
	newUser.Age = 0       // 0으로 초기화
	newUser.Email = ""    // 빈 문자열로 초기화

	// DB에 유저 삽입
	insertedID, err := app.Models.InsertUser(newUser.Name, newUser.Nickname, newUser.SnsID, newUser.Gender, newUser.Age, newUser.SnsType, newUser.Email)
	if err != nil {
		http.Error(w, "Failed to insert user", http.StatusInternalServerError)
		return
	}

	newUser.ID = int(insertedID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newUser)
}

// 유저 정보 업데이트
func (app *Config) updateUser(w http.ResponseWriter, r *http.Request) {
	// URL에서 유저 ID 가져오기
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		log.Printf("Failed to Atoi user ID, err: %s", err.Error())
		http.Error(w, "Failed to Atoi user ID", http.StatusInternalServerError)
		return
	}

	var updatedUser User

	// 요청에서 유저 데이터를 읽음
	err = json.NewDecoder(r.Body).Decode(&updatedUser)
	if err != nil {
		log.Printf("Body: %v, err: %s", updatedUser, err.Error())
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	updatedUser.ID = userID

	// DB에서 유저 정보 업데이트
	err = app.Models.UpdateUser(updatedUser.ID, updatedUser.Name, updatedUser.Nickname, updatedUser.Gender, updatedUser.Age)
	if err != nil {
		log.Printf("Failed to update user, user: %v, err: %s", updatedUser, err.Error())
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
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		log.Printf("Failed to Atoi user ID, err: %s", err.Error())
		http.Error(w, "Failed to Atoi user ID", http.StatusInternalServerError)
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
}
