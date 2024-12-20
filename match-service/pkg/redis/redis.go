package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/baaami/dorandoran/match-service/pkg/event"
	"github.com/baaami/dorandoran/match-service/pkg/types"
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

// MonitorAndMatchUsers monitors Redis queues and matches users
func (r *RedisClient) MonitorAndMatchUsers(coupleCount int, emitter *event.Emitter) error {
	maleQueueKey := fmt.Sprintf("matching_queue_male_%d", coupleCount)
	femaleQueueKey := fmt.Sprintf("matching_queue_female_%d", coupleCount)

	// 남성과 여성 대기열에서 사용자 확인
	maleUsers, err := r.Client.LRange(ctx, maleQueueKey, 0, int64(coupleCount-1)).Result()
	if err != nil {
		return fmt.Errorf("failed to retrieve male users from %s: %w", maleQueueKey, err)
	}
	femaleUsers, err := r.Client.LRange(ctx, femaleQueueKey, 0, int64(coupleCount-1)).Result()
	if err != nil {
		return fmt.Errorf("failed to retrieve female users from %s: %w", femaleQueueKey, err)
	}

	// 매칭 조건 확인
	if len(maleUsers) >= coupleCount && len(femaleUsers) >= coupleCount {
		var matchedMales, matchedFemales []types.WaitingUser

		// 남성 사용자 파싱
		for _, maleData := range maleUsers {
			var user types.WaitingUser
			if err := json.Unmarshal([]byte(maleData), &user); err != nil {
				log.Printf("Failed to unmarshal male user: %v", err)
				continue
			}
			matchedMales = append(matchedMales, user)
		}

		// 여성 사용자 파싱
		for _, femaleData := range femaleUsers {
			var user types.WaitingUser
			if err := json.Unmarshal([]byte(femaleData), &user); err != nil {
				log.Printf("Failed to unmarshal female user: %v", err)
				continue
			}
			matchedFemales = append(matchedFemales, user)
		}

		// 매칭 ID 생성
		matchID := generateMatchID(matchedMales, matchedFemales)

		// 매칭 이벤트 생성
		matchEvent := types.MatchEvent{
			MatchId:      matchID,
			MatchedUsers: append(matchedMales, matchedFemales...),
		}

		// 매칭 이벤트 발행
		err := emitter.PublishMatchEvent(matchEvent)
		if err != nil {
			log.Printf("Failed to publish match event for match ID %s: %v", matchID, err)
		}

		// 매칭된 사용자들을 큐에서 제거
		for _, userData := range maleUsers[:coupleCount] {
			log.Printf("matched male: %s", userData)
			r.Client.LPop(ctx, maleQueueKey)
		}
		for _, userData := range femaleUsers[:coupleCount] {
			log.Printf("matched female: %s", userData)
			r.Client.LPop(ctx, femaleQueueKey)
		}
	}

	return nil
}

// generateMatchID creates a unique match ID based on datetime and user IDs
func generateMatchID(males, females []types.WaitingUser) string {
	timestamp := time.Now().Format("20060102150405")
	var userIDs []string
	for _, user := range append(males, females...) {
		userIDs = append(userIDs, strconv.Itoa(user.ID))
	}
	return fmt.Sprintf("%s_%s", timestamp, joinIDs(userIDs))
}

func joinIDs(ids []string) string {
	return strings.Join(ids, "_")
}
