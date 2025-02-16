package redis

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/go-redis/redis/v8"
)

func (r *RedisClient) JoinRoom(roomID string, userID int) error {
	roomKey := fmt.Sprintf("join_room:%s", roomID)
	err := r.Client.SAdd(ctx, roomKey, strconv.Itoa(userID)).Err()
	if err != nil {
		return fmt.Errorf("failed to join room %s for user %d: %v", roomID, userID, err)
	}
	log.Printf("User %d joined room %s", userID, roomID)
	return nil
}

func (r *RedisClient) LeaveRoom(roomID string, userID int) error {
	roomKey := fmt.Sprintf("join_room:%s", roomID)
	err := r.Client.SRem(ctx, roomKey, strconv.Itoa(userID)).Err()
	if err != nil {
		return fmt.Errorf("❌ Redis LeaveRoom 실패: %w", err)
	}
	log.Printf("🚪 User %d removed from room %s in Redis", userID, roomID)
	return nil
}

// Redis에서 모든 Room ID 가져오기
func (r *RedisClient) GetAllRoomsFromRedis() ([]string, error) {
	ctx := context.Background()

	// Redis에서 방 목록 가져오기
	roomIDs, err := r.Client.SMembers(ctx, "rooms:list").Result()
	if err != nil {
		log.Printf("Failed to get room list from Redis: %v", err)
		return nil, err
	}

	return roomIDs, nil
}

// Redis에서 채팅방 남은 시간 가져오기
func (r *RedisClient) GetRoomRemainingTime(roomID string) (int, error) {
	ctx := context.Background()

	ttl, err := r.Client.TTL(ctx, roomID).Result()
	if err != nil {
		log.Printf("Failed to get remaining time for RoomID %s: %v", roomID, err)
		return 0, err
	}

	if ttl <= 0 {
		log.Printf("RoomID %s has no remaining time or is expired", roomID)
		return 0, nil // 타임아웃이 만료된 경우
	}

	return int(ttl.Seconds()), nil
}

func (r *RedisClient) GetRoomStatus(roomID string) (int, error) {
	statusKey := fmt.Sprintf("room_status:%s", roomID)
	statusStr, err := r.Client.Get(ctx, statusKey).Result()
	if err == redis.Nil {
		return 0, fmt.Errorf("status not found for room %s", roomID) // 상태가 없으면 에러 반환
	} else if err != nil {
		return 0, fmt.Errorf("failed to get status for room %s: %v", roomID, err)
	}

	status, convErr := strconv.Atoi(statusStr)
	if convErr != nil {
		return 0, fmt.Errorf("failed to convert status for room %s: %v", roomID, convErr)
	}

	log.Printf("Get status for room %s: %d", roomID, status)
	return status, nil
}
