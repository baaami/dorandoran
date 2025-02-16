package redis

import (
	"context"
	"log"
)

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
