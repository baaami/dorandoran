package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/baaami/dorandoran/user/cmd/data"
)

// 유저 정보 조회
func (app *Config) findUser(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("Missing parameters: sns_type=%s, sns_id=%s", snsType, snsID)
		http.Error(w, "Missing sns_type or sns_id", http.StatusBadRequest)
		return
	}

	// sns_type을 정수로 변환
	nSnsType, err := strconv.Atoi(snsType)
	if err != nil {
		log.Printf("Invalid sns_type parameter: %s, error: %v", snsType, err)
		http.Error(w, fmt.Sprintf("Bad Parameter sns_type: %s", snsType), http.StatusBadRequest)
		return
	}

	// sns_id를 정수로 변환
	nSnsID, err := strconv.ParseInt(snsID, 10, 64)
	if err != nil {
		log.Printf("Invalid sns_id parameter: %s, error: %v", snsID, err)
		http.Error(w, fmt.Sprintf("Bad Parameter sns_id: %s", snsID), http.StatusBadRequest)
		return
	}

	// DB에서 사용자 조회
	user, err := app.Models.GetUserBySNS(nSnsType, nSnsID)
	if err != nil {
		log.Printf("Error fetching user for sns_type=%d, sns_id=%d, error: %v", nSnsType, nSnsID, err)
		http.Error(w, "Error fetching user", http.StatusInternalServerError)
		return
	}

	// 유저가 존재하지 않는 경우
	if user == nil {
		log.Printf("User not found for sns_type=%d, sns_id=%d", nSnsType, nSnsID)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// 유저가 존재하는 경우, StatusOK와 함께 유저 정보 반환
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(*user); err != nil {
		log.Printf("Error encoding user data: %v", err)
		http.Error(w, "Error encoding user data", http.StatusInternalServerError)
	}
}

// 유저 정보 삽입
func (app *Config) registerUser(w http.ResponseWriter, r *http.Request) {
	var newUser data.User

	// 요청에서 유저 데이터를 읽음
	err := json.NewDecoder(r.Body).Decode(&newUser)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// 유저 정보 로그 출력
	log.Printf("Registering user with the following details: %v", newUser)

	// DB에 유저 삽입
	insertedID, err := app.Models.InsertUser(newUser)
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

	var updatedUser data.User

	// 요청에서 유저 데이터를 읽음
	err = json.NewDecoder(r.Body).Decode(&updatedUser)
	if err != nil {
		log.Printf("Body: %v, err: %s", updatedUser, err.Error())
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	updatedUser.ID = userID

	// DB에서 유저 정보 업데이트
	err = app.Models.UpdateUser(updatedUser)
	if err != nil {
		log.Printf("Failed to update user, user: %v, err: %s", updatedUser, err.Error())
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	// DB에서 유저 정보 획득
	user, err := app.Models.GetUserByID(updatedUser.ID)
	if err != nil {
		log.Printf("Failed to get updated user, err: %s", err.Error())
		http.Error(w, "Failed to get updated user", http.StatusInternalServerError)
		return
	}

	// 업데이트된 유저 정보 반환
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
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
