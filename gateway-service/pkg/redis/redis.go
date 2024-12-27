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

// GetSession: Redis에서 세션 조회
func (r *RedisClient) GetUserBySessionID(sessionID string) (int, error) {
	sUserID, err := r.Client.Get(ctx, sessionID).Result()
	if err == redis.Nil {
		log.Printf("sessionID is not exist in DB")
		return 0, fmt.Errorf("session not found")
	} else if err != nil {
		log.Printf("Get Session Error, %s", err.Error())
		return 0, err
	}

	userID, err := strconv.Atoi(sUserID)
	if err != nil {
		log.Printf("Failed to Atoi, user id: %s", sUserID)
		return 0, nil
	}

	return userID, nil
}
