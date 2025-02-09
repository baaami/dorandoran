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
