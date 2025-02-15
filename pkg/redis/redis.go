package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"solo/pkg/types/commontype"
	"time"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

type RedisClient struct {
	Client *redis.Client
}

// Redis 클라이언트 생성
func NewRedisClient() (*RedisClient, error) {
	// 환경 변수에서 Redis 설정 가져오기
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisPassword := os.Getenv("REDIS_PASSWORD")

	// Redis 클라이언트 설정
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		DB:       0, // 기본 DB 사용
	})

	// 연결 확인
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Printf("Failed to connect to Redis: %v", err)
		return nil, err
	}

	return &RedisClient{Client: rdb}, nil
}

// 데이터 저장
func (r *RedisClient) Set(key string, value interface{}, expiration time.Duration) error {
	err := r.Client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		log.Printf("Failed to set key %s in Redis: %v", key, err)
		return err
	}
	return nil
}

// 데이터 조회
func (r *RedisClient) Get(key string) (string, error) {
	val, err := r.Client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("key %s does not exist", key)
	} else if err != nil {
		log.Printf("Failed to get key %s from Redis: %v", key, err)
		return "", err
	}
	return val, nil
}

// 데이터 삭제
func (r *RedisClient) Delete(key string) error {
	err := r.Client.Del(ctx, key).Err()
	if err != nil {
		log.Printf("Failed to delete key %s from Redis: %v", key, err)
		return err
	}
	return nil
}

func (r *RedisClient) MonitorAndMatchUsers(coupleCount int) ([]commontype.WaitingUser, error) {
	maleQueueKey := fmt.Sprintf("matching_queue_male_%d", coupleCount)
	femaleQueueKey := fmt.Sprintf("matching_queue_female_%d", coupleCount)

	// 남성과 여성 대기열에서 사용자 확인
	maleUsers, err := r.Client.LRange(ctx, maleQueueKey, 0, int64(coupleCount-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve male users from %s: %w", maleQueueKey, err)
	}
	femaleUsers, err := r.Client.LRange(ctx, femaleQueueKey, 0, int64(coupleCount-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve female users from %s: %w", femaleQueueKey, err)
	}

	// 매칭 조건 확인
	if len(maleUsers) >= coupleCount && len(femaleUsers) >= coupleCount {
		var matchedUsers []commontype.WaitingUser

		// 남성 사용자 파싱
		for _, maleData := range maleUsers[:coupleCount] {
			var user commontype.WaitingUser
			if err := json.Unmarshal([]byte(maleData), &user); err != nil {
				log.Printf("❌ Failed to unmarshal male user: %v", err)
				continue
			}
			matchedUsers = append(matchedUsers, user)
		}

		// 여성 사용자 파싱
		for _, femaleData := range femaleUsers[:coupleCount] {
			var user commontype.WaitingUser
			if err := json.Unmarshal([]byte(femaleData), &user); err != nil {
				log.Printf("❌ Failed to unmarshal female user: %v", err)
				continue
			}
			matchedUsers = append(matchedUsers, user)
		}

		// 매칭된 사용자들을 큐에서 제거
		for i := 0; i < coupleCount; i++ {
			r.Client.LPop(ctx, maleQueueKey)
			r.Client.LPop(ctx, femaleQueueKey)
		}

		log.Printf("✅ Successfully matched %d males and %d females", coupleCount, coupleCount)
		return matchedUsers, nil
	}

	return nil, nil
}
