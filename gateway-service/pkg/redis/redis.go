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

func NewRedisClient() *RedisClient {
	// 환경 변수에서 Redis 설정 불러오기
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisPassword := os.Getenv("REDIS_PASSWORD")

	// Redis 클라이언트 생성
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		DB:       0,
	})

	// Redis 연결 확인
	_, err := client.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	} else {
		log.Println("Successfully connected to Redis")
	}

	return &RedisClient{
		Client: client,
	}
}

func (r *RedisClient) AddUserToQueue(userID string) error {
	return r.Client.LPush(ctx, "waiting_queue", userID).Err()
}

func (r *RedisClient) PopUsersFromQueue() (string, string, error) {
	// 두 명의 유저를 대기열에서 꺼냄
	user1, err := r.Client.RPop(ctx, "waiting_queue").Result()
	if err == redis.Nil {
		return "", "", nil // 유저 없음
	} else if err != nil {
		return "", "", err
	}

	user2, err := r.Client.RPop(ctx, "waiting_queue").Result()
	if err == redis.Nil {
		// 두 번째 유저가 없으면 첫 번째 유저를 다시 대기열에 넣음
		r.Client.LPush(ctx, "waiting_queue", user1)
		return "", "", nil // 매칭 불가
	} else if err != nil {
		return "", "", err
	}

	return user1, user2, nil
}

// GetAllUsersInQueue: 대기열에 있는 모든 유저 출력
func (r *RedisClient) GetAllUsersInQueue() ([]string, error) {
	users, err := r.Client.LRange(ctx, "waiting_queue", 0, -1).Result()
	if err != nil {
		log.Printf("Failed to get users from queue: %v", err)
		return nil, err
	}

	for i, user := range users {
		fmt.Printf("User %d: %s\n", i+1, user)
	}

	return users, nil
}
