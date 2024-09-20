package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

// 매칭 요청에 대한 POST 요청 처리
func (app *Config) addUserToQueue(w http.ResponseWriter, r *http.Request) {
	type MatchRequest struct {
		UserID string `json:"user_id"`
	}

	var req MatchRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.UserID == "" {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Redis 대기열에 유저 추가
	err = app.RedisClient.LPush(ctx, "waiting_queue", req.UserID).Err()
	if err != nil {
		log.Printf("Failed to add user to queue: %v", err)
		http.Error(w, "Failed to add user to queue", http.StatusInternalServerError)
		return
	}

	// 성공 응답
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("User added to queue"))
}

// 매칭 상태 확인 GET 요청 처리
func (app *Config) getMatchStatus(w http.ResponseWriter, r *http.Request) {
	// Redis에서 두 명의 유저를 대기열에서 꺼내서 매칭
	user1, err := app.RedisClient.RPop(ctx, "waiting_queue").Result()
	if err == redis.Nil {
		// 대기열에 유저가 없는 경우
		w.WriteHeader(http.StatusNoContent)
		return
	} else if err != nil {
		log.Printf("Failed to pop user from queue: %v", err)
		http.Error(w, "Failed to pop user from queue", http.StatusInternalServerError)
		return
	}

	// 두 번째 유저도 대기열에서 꺼내서 매칭
	user2, err := app.RedisClient.RPop(ctx, "waiting_queue").Result()
	if err == redis.Nil {
		// 두 번째 유저가 없으면 첫 번째 유저를 다시 대기열에 넣음
		app.RedisClient.LPush(ctx, "waiting_queue", user1)
		w.WriteHeader(http.StatusNoContent)
		return
	} else if err != nil {
		log.Printf("Failed to pop second user from queue: %v", err)
		http.Error(w, "Failed to pop second user from queue", http.StatusInternalServerError)
		return
	}

	// 매칭 성공 시 room ID 생성
	roomID := user1 + "-" + user2

	// 매칭된 유저 정보를 응답
	type MatchResponse struct {
		User1ID string `json:"user1_id"`
		User2ID string `json:"user2_id"`
		RoomID  string `json:"room_id"`
	}

	response := MatchResponse{
		User1ID: user1,
		User2ID: user2,
		RoomID:  roomID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
