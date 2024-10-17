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

// AddUserToQueue: 대기열에 유저 추가
func (r *RedisClient) AddUserToQueue(userID string, coupleCnt int) error {
	// TODO: 1:1 ~ 4:4 대기열 큐를 각기 다르게 생성해야함
	err := r.Client.LPush(ctx, "waiting_queue", userID).Err()
	if err != nil {
		log.Printf("Failed to add user to queue: %v", err)
		return err
	}
	log.Printf("User %s added to Redis matching queue", userID)
	return nil
}

// TODO: 존재하는 모든 대기열에 대해서 모니터링 해야함
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

	if len(users) < n {
		// Pop된 유저가 부족하면 다시 대기열에 삽입
		for _, user := range users {
			err := r.Client.RPush(ctx, "waiting_queue", user).Err()
			if err != nil {
				return nil, err
			}
		}
		return []string{}, nil
	}

	return users, nil
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
