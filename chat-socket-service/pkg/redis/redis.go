package redis

import (
	"context"
	"fmt"
	"log"
	"os"

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

// 활성 사용자 등록
func (r *RedisClient) RegisterActiveUser(userID, serverID string) error {
	return r.Client.HSet(ctx, "client:active", userID, serverID).Err()
}

// 활성 사용자 제거
func (r *RedisClient) UnregisterActiveUser(userID string) error {
	return r.Client.HDel(ctx, "client:active", userID).Err()
}

// 사용자 활성 상태 확인
func (r *RedisClient) IsUserActive(userID string) (bool, error) {
	serverID, err := r.Client.HGet(ctx, "client:active", userID).Result()
	if err == redis.Nil {
		return false, nil // 활성화되지 않은 사용자
	} else if err != nil {
		return false, fmt.Errorf("failed to check user active status: %v", err)
	}

	return serverID != "", nil
}

// Room의 활성 사용자 ID 리스트를 반환
func (r *RedisClient) GetActiveUserIDs(roomID string) ([]string, error) {
	// Step 1: Room의 사용자 ID 리스트 가져오기
	roomKey := fmt.Sprintf("room:%s", roomID)
	userIDs, err := r.Client.SMembers(ctx, roomKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get users for room %s: %v", roomID, err)
	}

	log.Printf("ROOM (%s) MEMBER in Redis, users: %v", roomID, userIDs)

	// Step 2: 활성 사용자 필터링
	activeUsers := []string{}
	for _, userID := range userIDs {
		activeKey := "client:active"
		active, err := r.Client.HGet(ctx, activeKey, userID).Result()
		if err == redis.Nil || active != "unique-server-id" {
			log.Printf("active: %s", active)
			// 사용자 비활성화 상태일 경우 무시
			continue
		} else if err != nil {
			return nil, fmt.Errorf("failed to check active status for user %s: %v", userID, err)
		}

		// 활성 사용자 추가
		activeUsers = append(activeUsers, userID)
	}

	log.Printf("ACTIVE USER in Redis, users: %s", activeUsers)

	return activeUsers, nil
}

func (r *RedisClient) JoinRoom(roomID, userID string) error {
	roomKey := fmt.Sprintf("join_room:%s", roomID)
	err := r.Client.SAdd(ctx, roomKey, userID).Err()
	if err != nil {
		return fmt.Errorf("failed to join room %s for user %s: %v", roomID, userID, err)
	}
	log.Printf("User %s joined room %s", userID, roomID)
	return nil
}

func (r *RedisClient) LeaveRoom(roomID, userID string) error {
	roomKey := fmt.Sprintf("join_room:%s", roomID)
	err := r.Client.SRem(ctx, roomKey, userID).Err()
	if err != nil {
		return fmt.Errorf("failed to leave room %s for user %s: %v", roomID, userID, err)
	}
	log.Printf("User %s left room %s", userID, roomID)
	return nil
}

func (r *RedisClient) GetJoinedUser(roomID string) ([]string, error) {
	roomKey := fmt.Sprintf("join_room:%s", roomID)
	userIDs, err := r.Client.SMembers(ctx, roomKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get users for room %s: %v", roomID, err)
	}

	log.Printf("Users in room %s: %v", roomID, userIDs)
	return userIDs, nil
}
