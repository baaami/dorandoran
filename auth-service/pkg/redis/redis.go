package redis

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
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

// setSession: Redis에 세션 저장
func (r *RedisClient) setSession(sessionID string, userID string, expiresAt time.Duration) error {
	err := r.Client.Set(ctx, sessionID, userID, expiresAt).Err()
	if err != nil {
		log.Printf("Failed to set session in Redis: %v", err)
		return err
	}

	err = r.Client.Set(ctx, userID, sessionID, expiresAt).Err()
	if err != nil {
		log.Printf("Failed to set session in Redis: %v", err)
		return err
	}
	return nil
}

// 세션 생성 함수 (Redis에 세션 저장)
func (r *RedisClient) CreateSession(userID int) string {
	// 고유한 세션 ID 생성 (UUID 사용)
	sessionID := uuid.New().String()

	// 세션 만료 시간 설정 (예: 24시간)
	expiresAt := time.Hour * 24

	// Redis에 세션 저장
	err := r.setSession(sessionID, strconv.Itoa(userID), expiresAt)
	if err != nil {
		log.Printf("Failed to store session in Redis: %v", err)
	}

	// 생성된 세션 ID 반환
	return sessionID
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

// GetSessionByUserID: 사용자 ID로 세션 ID 조회
func (r *RedisClient) GetSessionByUserID(userID int) (string, error) {
	// Redis Hash에서 userID에 해당하는 sessionID를 조회
	sessionID, err := r.Client.Get(ctx, strconv.Itoa(userID)).Result()
	if err == redis.Nil {
		// 해당 userID가 없는 경우 처리
		log.Printf("not found user in db, userID: %d, err: %v", userID, err)
		return "", fmt.Errorf("not found user in db, userID: %d", userID)
	} else if err != nil {
		// Redis 오류 처리
		log.Printf("failed to get session by userID, userID: %d, err: %v", userID, err)
		return "", fmt.Errorf("failed to get session by userID: %v", err)
	}

	// 세션 ID 반환
	return sessionID, nil
}

// TODO: Session Delete 여부 확인 필요
// // DeleteSession: Redis에서 세션 삭제
// func (r *RedisClient) DeleteSession(userID int) error {
// 	sessionID, err := r.GetSessionByUserID(userID)
// 	if err != nil {
// 		log.Printf("Failed to GetSessionByUserID, user id: %d, err: %v", userID, err)
// 		return err
// 	}

// 	err = r.Client.Del(ctx, sessionID).Err()
// 	if err != nil {
// 		log.Printf("Failed to delete user in Redis, user id: %d, err: %v", userID, err)
// 		return err
// 	}

// 	err = r.Client.Del(ctx, userID).Err()
// 	if err != nil {
// 		log.Printf("Failed to delete user in Redis, user id: %s, err: %v", userID, err)
// 		return err
// 	}
// 	return nil
// }
