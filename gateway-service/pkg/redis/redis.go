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

// AddUserToQueue: 대기열에 유저 추가 (coupleCnt에 따른 queue 분리)
func (r *RedisClient) AddUserToQueue(userID string, coupleCnt int) error {
	queueName := fmt.Sprintf("matching_queue_%d", coupleCnt) // coupleCnt에 따른 대기열 이름 생성
	err := r.Client.LPush(ctx, queueName, userID).Err()
	if err != nil {
		log.Printf("Failed to add user to queue: %v", err)
		return err
	}
	log.Printf("User %s added to Redis matching queue %s", userID, queueName)
	return nil
}

// PopNUsersFromQueue: 특정 대기열에서 n명의 유저를 pop
func (r *RedisClient) PopNUsersFromQueue(coupleCnt, n int) ([]string, error) {
	queueName := fmt.Sprintf("matching_queue_%d", coupleCnt) // coupleCnt에 따른 대기열 이름
	var users []string
	for i := 0; i < n; i++ {
		user, err := r.Client.RPop(ctx, queueName).Result()
		if err == redis.Nil {
			break
		} else if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	// 매칭 인원이 부족하면 pop된 유저들을 다시 대기열에 삽입
	if len(users) < n {
		for _, user := range users {
			err := r.Client.RPush(ctx, queueName, user).Err()
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
