package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"solo/pkg/dto"
	"solo/services/user/service"
	"strconv"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// 유저 리스트 조회
func (h *UserHandler) FindUserList(w http.ResponseWriter, r *http.Request) {
	users, err := h.userService.GetUserList()
	if err != nil {
		http.Error(w, "Failed to retrieve user list", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(users)
}

// 특정 유저 조회
func (h *UserHandler) FindUser(w http.ResponseWriter, r *http.Request) {
	xUserID := r.Header.Get("X-User-ID")
	if xUserID == "" {
		http.Error(w, "User ID is required", http.StatusUnauthorized)
		return
	}

	userID, err := strconv.Atoi(xUserID)
	if err != nil {
		http.Error(w, fmt.Sprintf("User ID is not number, xUserID: %s", xUserID), http.StatusUnauthorized)
		return
	}

	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(user)
}

// 존재하는 유저인지 조회
func (h *UserHandler) CheckUser(w http.ResponseWriter, r *http.Request) {
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

	// DB에서 사용자 조회
	user, err := h.userService.GetUserBySNS(nSnsType, snsID)
	if err != nil {
		log.Printf("Error fetching user for sns_type=%d, sns_id=%s, error: %v", nSnsType, snsID, err)
		http.Error(w, "Error fetching user", http.StatusInternalServerError)
		return
	}

	// 유저가 존재하지 않는 경우
	if user == nil {
		log.Printf("User not found for sns_type=%d, sns_id=%s", nSnsType, snsID)
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

// 유저 등록
func (h *UserHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var newUser dto.UserDTO
	if err := json.NewDecoder(r.Body).Decode(&newUser); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	user, err := h.userService.RegisterUser(newUser)
	if err != nil {
		http.Error(w, "Failed to insert user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// 유저 업데이트
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	xUserID := r.Header.Get("X-User-ID")
	if xUserID == "" {
		http.Error(w, "User ID is required", http.StatusUnauthorized)
		return
	}

	userID, err := strconv.Atoi(xUserID)
	if err != nil {
		http.Error(w, fmt.Sprintf("User ID is not number, xUserID: %s", xUserID), http.StatusUnauthorized)
		return
	}

	var updatedUser dto.UserDTO
	if err := json.NewDecoder(r.Body).Decode(&updatedUser); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	updatedUser.ID = userID

	err = h.userService.UpdateUser(updatedUser)
	if err != nil {
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// 유저 삭제
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	xUserID := r.Header.Get("X-User-ID")
	if xUserID == "" {
		http.Error(w, "User ID is required", http.StatusUnauthorized)
		return
	}

	userID, err := strconv.Atoi(xUserID)
	if err != nil {
		http.Error(w, fmt.Sprintf("User ID is not number, xUserID: %s", xUserID), http.StatusUnauthorized)
		return
	}

	err = h.userService.DeleteUser(userID)
	if err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
