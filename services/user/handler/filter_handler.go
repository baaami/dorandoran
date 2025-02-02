package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"solo/pkg/dto"
	"solo/services/user/service"
	"strconv"
)

type FilterHandler struct {
	filterService *service.FilterService
}

func NewfilterHandler(filterService *service.FilterService) *FilterHandler {
	return &FilterHandler{
		filterService: filterService,
	}
}

// 특정 유저의 매칭 필터 조회
func (h *FilterHandler) FindMatchFilter(w http.ResponseWriter, r *http.Request) {
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

	filter, err := h.filterService.GetMatchFilterByUserID(userID)
	if err != nil {
		http.Error(w, "Failed to retrieve match filter", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(filter)
}

// 특정 유저의 매칭 필터 업데이트
func (h *FilterHandler) UpdateMatchFilter(w http.ResponseWriter, r *http.Request) {
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

	var filter dto.MatchFilterDTO
	if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	err = h.filterService.UpdateMatchFilter(userID, filter)
	if err != nil {
		http.Error(w, "Failed to update match filter", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
