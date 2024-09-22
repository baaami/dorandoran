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

// AddUserToQueue: 대기열에 유저 추가
func (r *RedisClient) AddUserToQueue(userID string) error {
	err := r.Client.LPush(ctx, "waiting_queue", userID).Err()
	if err != nil {
		log.Printf("Failed to add user to queue: %v", err)
		return err
	}
	log.Printf("User %s added to Redis matching queue", userID)
	return nil
}

// PopUsersFromQueue: 대기열에서 두 명의 유저를 가져옴
func (r *RedisClient) PopUsersFromQueue() (string, string, error) {
	user1, err := r.Client.RPop(ctx, "waiting_queue").Result()
	if err == redis.Nil {
		return "", "", nil
	} else if err != nil {
		log.Printf("Error popping user from queue: %v", err)
		return "", "", err
	}

	user2, err := r.Client.RPop(ctx, "waiting_queue").Result()
	if err == redis.Nil {
		// 유저가 하나만 있을 때 다시 큐에 넣기
		r.Client.LPush(ctx, "waiting_queue", user1)
		log.Printf("Only one user in queue. Re-adding %s", user1)
		return "", "", nil
	} else if err != nil {
		log.Printf("Error popping second user from queue: %v", err)
		return "", "", err
	}

	return user1, user2, nil
}
