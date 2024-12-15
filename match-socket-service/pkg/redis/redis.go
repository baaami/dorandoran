package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/baaami/dorandoran/match-socket-service/pkg/types"
	"github.com/go-redis/redis/v8"
	"github.com/samber/lo"
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

// AddUserToQueue adds a WaitingUser to the Redis set
func (r *RedisClient) AddUserToQueue(user types.WaitingUser) error {
	key := "matching_users"

	// Serialize WaitingUser to JSON
	userData, err := json.Marshal(user)
	if err != nil {
		log.Printf("Failed to serialize WaitingUser for user %d: %v", user.ID, err)
		return err
	}

	// Add serialized user data to Redis set
	if err := r.Client.SAdd(ctx, key, userData).Err(); err != nil {
		log.Printf("Failed to add user %d to Redis set: %v", user.ID, err)
		return err
	}
	log.Printf("User %d added to Redis set", user.ID)
	return nil
}

// RemoveUserFromQueue removes a WaitingUser from the Redis set
func (r *RedisClient) RemoveUserFromQueue(user types.WaitingUser) error {
	key := "matching_users"

	// Serialize WaitingUser to JSON
	userData, err := json.Marshal(user)
	if err != nil {
		log.Printf("Failed to serialize WaitingUser for user %d: %v", user.ID, err)
		return err
	}

	// Remove serialized user data from Redis set
	if err := r.Client.SRem(ctx, key, userData).Err(); err != nil {
		log.Printf("Failed to remove user %d from Redis set: %v", user.ID, err)
		return err
	}
	log.Printf("User %d removed from Redis set", user.ID)
	return nil
}

// IsUserInQueue checks if a WaitingUser exists in the Redis set
func (r *RedisClient) IsUserInQueue(user types.WaitingUser) (bool, error) {
	key := "matching_users"

	// Serialize WaitingUser to JSON
	userData, err := json.Marshal(user)
	if err != nil {
		log.Printf("Failed to serialize WaitingUser for user %d: %v", user.ID, err)
		return false, err
	}

	// Check if serialized user data exists in Redis set
	exists, err := r.Client.SIsMember(ctx, key, userData).Result()
	if err != nil {
		log.Printf("Failed to check if user %d exists in Redis set: %v", user.ID, err)
		return false, err
	}
	return exists, nil
}

// TODO: 다음 함수는 별도 1개의 서비스에서 동작하여 RabbitMQ 이벤트를 발행하도록
// 대기열을 모니터링하고 매칭된 사용자를 추출하는 함수
func (r *RedisClient) MonitorAndPopMatchingUsers(coupleCnt int) ([]string, error) {
	queueName := fmt.Sprintf("matching_queue_%d", coupleCnt)
	var maleMatches, femaleMatches []types.WaitingUser

	// Redis에서 전체 데이터를 가져옴
	usersJson, err := r.Client.LRange(ctx, queueName, 0, -1).Result()
	if err != nil {
		log.Printf("Failed to retrieve users from queue %s: %v", queueName, err)
		return nil, err
	}
	if len(usersJson) == 0 {
		return nil, nil
	}

	for _, userJson := range usersJson {
		var user types.WaitingUser
		err = json.Unmarshal([]byte(userJson), &user)
		if err != nil {
			log.Printf("Failed to unmarshal user data: %v", err)
			continue
		}

		// 성별에 따라 리스트에 추가
		if user.Gender == 0 {
			maleMatches = append(maleMatches, user)
		} else if user.Gender == 1 {
			femaleMatches = append(femaleMatches, user)
		}

		// 조건에 맞는지 확인
		if len(maleMatches) >= coupleCnt && len(femaleMatches) >= coupleCnt {
			selectedMales := maleMatches[:coupleCnt]
			selectedFemales := femaleMatches[:coupleCnt]
			allMatch := true

			// 모든 남성-여성 조합에 대해 `isMatching` 검사
			for _, male := range selectedMales {
				for _, female := range selectedFemales {
					if !isMatching(male, female) {
						allMatch = false
						break
					}
				}
				if !allMatch {
					break
				}
			}

			if allMatch {
				matchUserIdList := []string{}
				matchUserIdList = append(matchUserIdList, lo.Map(selectedMales, func(matchUser types.WaitingUser, index int) string {
					return strconv.Itoa(matchUser.ID)
				})...)
				matchUserIdList = append(matchUserIdList, lo.Map(selectedFemales, func(matchUser types.WaitingUser, index int) string {
					return strconv.Itoa(matchUser.ID)
				})...)

				// 매칭된 사용자를 큐에서 제거
				for _, matchedUser := range append(selectedMales, selectedFemales...) {
					userData, _ := json.Marshal(matchedUser)
					r.Client.LRem(ctx, queueName, 1, userData)
				}

				log.Printf("Successfully matched users: %v from queue %s", matchUserIdList, queueName)
				return matchUserIdList, nil
			} else {
				// 조건을 만족하지 않는 경우 매칭된 사용자들을 다시 큐에 추가
				log.Printf("No complete matching users found in queue %s that meet all conditions", queueName)
				for _, user := range append(selectedMales, selectedFemales...) {
					userData, _ := json.Marshal(user)
					r.Client.RPush(ctx, queueName, userData)
				}
				maleMatches = maleMatches[coupleCnt:]     // 앞부분 제거하고 남은 부분 유지
				femaleMatches = femaleMatches[coupleCnt:] // 앞부분 제거하고 남은 부분 유지
			}
		}
	}

	return nil, nil
}

// isMatching: 두 사용자가 매칭 조건에 맞는지 검사하는 함수
func isMatching(user1, user2 types.WaitingUser) bool {
	birthYear1, _ := strconv.Atoi(user1.Birth[:4])
	birthYear2, _ := strconv.Atoi(user2.Birth[:4])
	ageDifference := birthYear1 - birthYear2
	if ageDifference < 0 {
		ageDifference = -ageDifference
	}

	// 나이 조건 확인 (한 명이라도 나이 조건을 적용하지 않으면 매칭 가능)
	if user1.AgeGroupUse || user2.AgeGroupUse {
		if ageDifference > 3 {
			return false
		}
	}

	// 지역 조건 확인 (한 명이라도 지역 조건을 적용하지 않으면 매칭 가능)
	if user1.AddressRangeUse || user2.AddressRangeUse {
		if user1.Address.City != user2.Address.City {
			return false
		}
	}

	return true
}
