package redis

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

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

func (r *RedisClient) SetRoomStatus(roomID string, status int) error {
	statusKey := fmt.Sprintf("room_status:%s", roomID)
	err := r.Client.Set(ctx, statusKey, status, 0).Err() // 만료 시간 없음
	if err != nil {
		return fmt.Errorf("failed to set status for room %s: %v", roomID, err)
	}
	log.Printf("Set status for room %s to %d", roomID, status)
	return nil
}

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
