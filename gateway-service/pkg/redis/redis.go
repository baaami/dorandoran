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

// AddUserToQueue: 성별 및 매칭 타입에 따라 대기열에 사용자 추가
func (r *RedisClient) AddUserToQueue(queueName, userID string, birthYear int) error {
	// 나이 순으로 ZSET에 추가
	err := r.Client.ZAdd(ctx, queueName, &redis.Z{
		Score:  float64(birthYear),
		Member: userID,
	}).Err()
	if err != nil {
		log.Printf("Failed to add user to ZSET queue: %v", err)
		return err
	}

	// 순서 유지를 위해 List에도 추가
	err = r.Client.RPush(ctx, queueName+"_order", userID).Err()
	if err != nil {
		log.Printf("Failed to add user to List queue: %v", err)
		return err
	}

	log.Printf("User %s added to Redis matching queue %s", userID, queueName)
	return nil
}

func (r *RedisClient) PopNUsersByYearRange(queueName string, matchNum int, minYear, maxYear int) ([]string, error) {
	var userIDList []string

	// ±5년 범위 내에서 사용자 추출
	users, err := r.Client.ZRangeByScore(ctx, queueName, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", minYear),
		Max: fmt.Sprintf("%d", maxYear),
	}).Result()

	// 인원이 부족한 경우 빈 배열 반환
	if err != nil || len(users) < matchNum {
		return []string{}, nil // 부족한 경우 빈 결과 반환
	}

	// 필요한 인원만큼 추출하고, 나머지는 유지
	userIDList = users[:matchNum]

	// 대기열에서 매칭된 사용자 제거
	for _, userID := range userIDList {
		_, err := r.Client.ZRem(ctx, queueName, userID).Result()
		if err != nil {
			log.Printf("Failed to remove user %s from queue %s: %v", userID, queueName, err)
			return []string{}, err
		}
	}

	log.Printf("Popped %d users from queue %s", len(userIDList), queueName)
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

// ZSET 대기열에서 가장 오래된 연도와 가장 최근 연도를 반환
func (r *RedisClient) GetOldestAndYoungestYear(queueName string) (int, int, error) {
	// 가장 오래된 연도 가져오기
	oldestUser, err := r.Client.ZRangeWithScores(ctx, queueName, 0, 0).Result()
	if err != nil || len(oldestUser) == 0 {
		return 0, 0, fmt.Errorf("no users in queue")
	}
	oldYear := int(oldestUser[0].Score)

	// 가장 최근 연도 가져오기
	youngestUser, err := r.Client.ZRangeWithScores(ctx, queueName, -1, -1).Result()
	if err != nil || len(youngestUser) == 0 {
		return 0, 0, fmt.Errorf("no users in queue")
	}
	youngYear := int(youngestUser[0].Score)

	return oldYear, youngYear, nil
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
