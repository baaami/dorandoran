package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/baaami/dorandoran/match-socket-service/pkg/types"
	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

type RedisClient struct {
	Client *redis.Client
}

// NewRedisClient: Redis 연결을 생성하는 함수
func NewRedisClient() (*RedisClient, error) {
	// 환경 변수에서 Redis 설정 가져오기
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisPassword := os.Getenv("REDIS_PASSWORD")

	// Redis 클라이언트 설정
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		DB:       0, // 기본 DB 사용
	})

	// Redis 연결 확인
	_, err := client.Ping(ctx).Result()
	if err != nil {
		log.Printf("Failed to connect to Redis: %v", err)
		return nil, err
	}

	log.Println("Successfully connected to Redis")
	return &RedisClient{Client: client}, nil
}

// 대기열에 사용자 추가
func (r *RedisClient) AddUserToQueue(user types.WaitingUser) error {
	queueKey := fmt.Sprintf("matching_queue_%s_%d", getGenderString(user.Gender), user.CoupleCount)

	// Serialize WaitingUser to JSON
	userData, err := json.Marshal(user)
	if err != nil {
		log.Printf("Failed to serialize WaitingUser for user %d: %v", user.ID, err)
		return err
	}

	// Add serialized user data to Redis list
	if err := r.Client.RPush(ctx, queueKey, userData).Err(); err != nil {
		log.Printf("Failed to add user %d to Redis queue %s: %v", user.ID, queueKey, err)
		return err
	}
	log.Printf("User %d added to Redis queue %s", user.ID, queueKey)
	return nil
}

// RemoveUserFromQueue removes a WaitingUser from the appropriate Redis queue
func (r *RedisClient) RemoveUserFromQueue(user types.WaitingUser) error {
	genderQueuePrefix := fmt.Sprintf("matching_queue_%s", getGenderString(user.Gender))

	// Serialize WaitingUser to JSON
	userData, err := json.Marshal(user)
	if err != nil {
		log.Printf("Failed to serialize WaitingUser for user %d: %v", user.ID, err)
		return err
	}

	// Iterate over all possible couple counts (2 to 6)
	for coupleCount := types.MATCH_COUNT_MIN; coupleCount <= types.MATCH_COUNT_MAX; coupleCount++ {
		queueKey := fmt.Sprintf("%s_%d", genderQueuePrefix, coupleCount)

		// Attempt to remove the user from the current queue
		if err := r.Client.LRem(ctx, queueKey, 1, userData).Err(); err != nil {
			log.Printf("Failed to remove user %d from queue %s: %v", user.ID, queueKey, err)
			continue
		}

		// Check if the user was successfully removed
		length, err := r.Client.LLen(ctx, queueKey).Result()
		if err == nil && length > 0 {
			log.Printf("User %d successfully removed from queue %s", user.ID, queueKey)
			return nil
		}
	}

	log.Printf("User %d not found in any queue", user.ID)
	return nil
}

// IsUserInQueue checks if a WaitingUser exists in any Redis queue
func (r *RedisClient) IsUserInQueue(user types.WaitingUser) (bool, string, error) {
	genderQueuePrefix := fmt.Sprintf("matching_queue_%s", getGenderString(user.Gender))

	// Serialize WaitingUser to JSON
	userData, err := json.Marshal(user)
	if err != nil {
		log.Printf("Failed to serialize WaitingUser for user %d: %v", user.ID, err)
		return false, "", err
	}

	// Iterate over all possible couple counts (2 to 6)
	for coupleCount := types.MATCH_COUNT_MIN; coupleCount <= types.MATCH_COUNT_MAX; coupleCount++ {
		queueKey := fmt.Sprintf("%s_%d", genderQueuePrefix, coupleCount)

		// Check if the user exists in the current queue
		exists, err := r.Client.LPos(ctx, queueKey, string(userData), redis.LPosArgs{}).Result()
		if err == nil && exists >= 0 {
			log.Printf("User %d found in queue %s", user.ID, queueKey)
			return true, queueKey, nil
		}
	}

	return false, "", nil
}

// getGenderString converts gender integer to string
func getGenderString(gender int) string {
	if gender == 0 {
		return "male"
	}
	return "female"
}
