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

func (r *RedisClient) PopNUsersFromQueue(n int) ([]string, error) {
	var users []string
	for i := 0; i < n; i++ {
		user, err := r.Client.RPop(ctx, "waiting_queue").Result()
		if err == redis.Nil {
			break
		} else if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

// GetSession: Redis에서 세션 조회
func (r *RedisClient) GetSession(sessionID string) (string, error) {
	snsID, err := r.Client.Get(ctx, sessionID).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("session not found")
	} else if err != nil {
		return "", err
	}
	return snsID, nil
}
