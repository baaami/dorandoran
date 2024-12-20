package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/baaami/dorandoran/broker/pkg/types"
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

// AddUserToQueue: 성별 및 매칭 타입에 따라 대기열에 사용자 추가
func (r *RedisClient) AddUserToQueue(coupleCnt int, waitingUser types.WaitingUser) error {
	queueName := fmt.Sprintf("matching_queue_%d", coupleCnt)
	waitingUserData, err := json.Marshal(waitingUser)
	if err != nil {
		log.Printf("Failed to marshal waitingUser data for waitingUser %d: %v", waitingUser.ID, err)
		return err
	}

	// 대기열에 사용자 정보 추가
	err = r.Client.RPush(ctx, queueName, waitingUserData).Err()
	if err != nil {
		log.Printf("Failed to add waitingUser to queue: %v", err)
		return err
	}

	log.Printf("User %d added to Redis matching queue %s", waitingUser.ID, queueName)
	return nil
}

// PopUserFromQueue: 특정 사용자를 대기열에서 제거하는 함수
func (r *RedisClient) PopUserFromQueue(userID int, coupleCnt int) error {
	queueName := fmt.Sprintf("matching_queue_%d", coupleCnt)
	queueLength, err := r.Client.LLen(ctx, queueName).Result()
	if err != nil {
		log.Printf("Failed to get queue length for %s: %v", queueName, err)
		return err
	}

	for i := 0; i < int(queueLength); i++ {
		userJson, err := r.Client.LIndex(ctx, queueName, int64(i)).Result()
		if err != nil {
			log.Printf("Failed to get user from queue %s: %v", queueName, err)
			continue
		}

		var user types.WaitingUser
		err = json.Unmarshal([]byte(userJson), &user)
		if err != nil {
			log.Printf("Failed to unmarshal user data: %v", err)
			continue
		}

		if user.ID == userID {
			_, err = r.Client.LRem(ctx, queueName, 1, userJson).Result()
			if err != nil {
				log.Printf("Failed to remove user %d from queue %s: %v", userID, queueName, err)
				return err
			}
			log.Printf("User %d removed from Redis matching queue %s", userID, queueName)
			return nil
		}
	}

	return nil
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
