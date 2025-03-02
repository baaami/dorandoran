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
	if redisHost == "" {
		redisHost = "doran-redis"
	}
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}
	redisPassword := os.Getenv("REDIS_PASSWORD")

	var rdb *redis.Client
	if redisPassword == "" {
		rdb = redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
			DB:   0, // 기본 DB 사용
		})
	} else {
		rdb = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
			Password: redisPassword,
			DB:       0, // 기본 DB 사용
		})
	}

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

// 채팅방 정보를 Redis에 추가
func (r *RedisClient) AddRoomToRedis(roomID string, userIDs []int, duration time.Duration) error {
	roomKey := fmt.Sprintf("room:%s", roomID)

	// 유저 ID들을 Redis Set에 저장
	strUserIDs := make([]interface{}, len(userIDs))
	for i, id := range userIDs {
		strUserIDs[i] = fmt.Sprintf("%d", id)
	}
	err := r.Client.SAdd(ctx, roomKey, strUserIDs...).Err()
	if err != nil {
		return fmt.Errorf("failed to add room %s to Redis: %v", roomID, err)
	}

	// // 만료 시간 설정
	// err = r.Client.Expire(ctx, roomKey, duration).Err()
	// if err != nil {
	// 	return fmt.Errorf("failed to set expiration for room %s: %v", roomID, err)
	// }

	log.Printf("Room %s added to Redis with users %v, expires in %v", roomID, userIDs, duration)
	return nil
}

// 채팅방 상태 설정
func (r *RedisClient) SetRoomStatus(roomID string, status int) error {
	statusKey := fmt.Sprintf("room_status:%s", roomID)
	err := r.Client.Set(ctx, statusKey, status, 0).Err() // 만료 시간 없음
	if err != nil {
		return fmt.Errorf("failed to set status for room %s: %v", roomID, err)
	}
	log.Printf("Set status for room %s to %d", roomID, status)
	return nil
}

// 채팅방 타임아웃 설정
func (r *RedisClient) SetRoomTimeout(roomID string, duration time.Duration) error {
	err := r.Client.Set(ctx, roomID, duration.Seconds(), duration).Err()
	if err != nil {
		log.Printf("Failed to set room timeout for RoomID %s: %v", roomID, err)
		return err
	}

	err = r.Client.SAdd(ctx, "rooms:list", roomID).Err()
	if err != nil {
		log.Printf("Failed to add RoomID %s to rooms list: %v", roomID, err)
		return err
	}

	log.Printf("Room timeout set for RoomID %s: %v seconds", roomID, duration.Seconds())
	return nil
}

func (r *RedisClient) RemoveRoomFromRedis(roomID string) error {
	ctx := context.Background()

	// Redis에서 방 제거
	err := r.Client.Del(ctx, roomID).Err()
	if err != nil {
		log.Printf("Failed to delete room %s from Redis: %v", roomID, err)
		return err
	}

	// 방 목록에서도 제거
	err = r.Client.SRem(ctx, "rooms:list", roomID).Err()
	if err != nil {
		log.Printf("Failed to remove RoomID %s from rooms list: %v", roomID, err)
		return err
	}

	log.Printf("RoomID %s removed from Redis", roomID)
	return nil
}
