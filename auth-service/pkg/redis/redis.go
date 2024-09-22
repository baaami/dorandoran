package redis

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

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

// SetSession: Redis에 세션 저장
func (r *RedisClient) SetSession(sessionID string, userID string, expiresAt time.Duration) error {
	err := r.Client.Set(ctx, sessionID, userID, expiresAt).Err()
	if err != nil {
		log.Printf("Failed to set session in Redis: %v", err)
		return err
	}
	return nil
}

// GetSession: Redis에서 세션 조회
func (r *RedisClient) GetSession(sessionID string) (string, error) {
	userID, err := r.Client.Get(ctx, sessionID).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("session not found")
	} else if err != nil {
		return "", err
	}
	return userID, nil
}

// DeleteSession: Redis에서 세션 삭제
func (r *RedisClient) DeleteSession(sessionID string) error {
	err := r.Client.Del(ctx, sessionID).Err()
	if err != nil {
		log.Printf("Failed to delete session in Redis: %v", err)
		return err
	}
	return nil
}

// GetSessionByUserID: 사용자 ID로 세션 ID 조회
func (r *RedisClient) GetSessionByUserID(userID string) (string, error) {
	sessionID, err := r.Client.Get(ctx, userID).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("no session found for user")
	} else if err != nil {
		return "", err
	}
	return sessionID, nil
}
