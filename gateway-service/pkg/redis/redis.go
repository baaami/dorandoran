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

// AddUserToQueue: 성별 및 coupleCnt에 따른 대기열에 유저 추가
func (r *RedisClient) AddUserToQueue(queueName, userID string) error {
	err := r.Client.LPush(ctx, queueName, userID).Err()
	if err != nil {
		log.Printf("Failed to add user to queue: %v", err)
		return err
	}
	log.Printf("User %s added to Redis matching queue %s", userID, queueName)
	return nil
}

// PopNUsersFromQueue: 특정 대기열에서 matchNum 명의 유저를 pop
func (r *RedisClient) PopNUsersFromQueue(queueName string, matchNum int) ([]string, error) {
	var userIDList []string
	for i := 0; i < matchNum; i++ {
		user, err := r.Client.RPop(ctx, queueName).Result()
		if err == redis.Nil {
			break
		} else if err != nil {
			return nil, err
		}
		userIDList = append(userIDList, user)
	}

	// 매칭 인원이 부족하면 pop된 유저들을 다시 대기열에 삽입
	if len(userIDList) < matchNum {
		for _, user := range userIDList {
			err := r.Client.RPush(ctx, queueName, user).Err()
			if err != nil {
				return nil, err
			}
		}
		return []string{}, nil
	}

	return userIDList, nil
}

// 특정 대기열에서 주어진 userID를 pop (매칭 중에 매칭을 종료할 경우 사용)
func (r *RedisClient) PopUserFromQueue(userID string, gender, coupleCnt int) (bool, error) {
	queueName := fmt.Sprintf("matching_queue_%d_%d", gender, coupleCnt) // coupleCnt에 따른 대기열 이름
	var popped bool

	// Redis 리스트의 길이를 먼저 구하고 대기열을 순회하면서 특정 userID를 pop
	queueLength, err := r.Client.LLen(ctx, queueName).Result()
	if err != nil {
		return false, err
	}

	for i := 0; i < int(queueLength); i++ {
		user, err := r.Client.LIndex(ctx, queueName, int64(i)).Result()
		if err == redis.Nil {
			break
		} else if err != nil {
			return false, err
		}

		if user == userID {
			// 특정 userID를 pop하기 위해 Redis 리스트에서 해당 인덱스의 값을 삭제
			_, err = r.Client.LRem(ctx, queueName, 1, userID).Result()
			if err != nil {
				return false, err
			}
			log.Printf("User %s removed from Redis matching queue %s", userID, queueName)
			popped = true
			break
		}
	}

	if !popped {
		log.Printf("User %s not found in Redis matching queue %s", userID, queueName)
	}

	return popped, nil
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
