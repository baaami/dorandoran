package redis

import (
	"fmt"
	"log"
	"strconv"

	"github.com/go-redis/redis/v8"
)

func (r *RedisClient) GetInActiveUserIDs(roomID string) ([]int, error) {
	// TODO: MongoDB 시작 시 Redis와 동기화 작업이 필요함
	roomKey := fmt.Sprintf("room:%s", roomID)
	userIDs, err := r.Client.SMembers(ctx, roomKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get users for room %s: %v", roomID, err)
	}

	log.Printf("ROOM (%s) MEMBER in Redis, users: %v", roomID, userIDs)

	inactiveUsers := []int{}
	for _, sUserID := range userIDs {
		activeKey := "client:active"
		active, err := r.Client.HGet(ctx, activeKey, sUserID).Result()
		if err == redis.Nil && active != "unique-server-id" {
			userID, err := strconv.Atoi(sUserID)
			if err != nil {
				log.Printf("sUserID is not number: %s", sUserID)
				continue
			}

			// 활성 사용자 추가
			inactiveUsers = append(inactiveUsers, userID)
		} else if err != nil {
			return nil, fmt.Errorf("failed to check active status for user %s: %v", sUserID, err)
		}
	}

	log.Printf("INACTIVE USER in Redis, users: %v", inactiveUsers)

	return inactiveUsers, nil
}
