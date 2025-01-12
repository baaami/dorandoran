package redis

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/baaami/dorandoran/chat-socket-service/pkg/types"
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

// 활성 사용자 등록
func (r *RedisClient) RegisterActiveUser(userID int, serverID string) error {
	return r.Client.HSet(ctx, "client:active", strconv.Itoa(userID), serverID).Err()
}

// 활성 사용자 제거
func (r *RedisClient) UnregisterActiveUser(userID int) error {
	return r.Client.HDel(ctx, "client:active", strconv.Itoa(userID)).Err()
}

// 사용자 활성 상태 확인
func (r *RedisClient) IsUserActive(userID int) (bool, error) {
	serverID, err := r.Client.HGet(ctx, "client:active", strconv.Itoa(userID)).Result()
	if err == redis.Nil {
		return false, nil // 활성화되지 않은 사용자
	} else if err != nil {
		return false, fmt.Errorf("failed to check user active status: %v", err)
	}

	return serverID != "", nil
}

// Room의 활성 사용자 ID 리스트를 반환
func (r *RedisClient) GetActiveUserIDs(roomID string) ([]int, error) {
	// Step 1: Room의 사용자 ID 리스트 가져오기
	// TODO: MongoDB 시작 시 Redis와 동기화 작업이 필요함
	roomKey := fmt.Sprintf("room:%s", roomID)
	userIDs, err := r.Client.SMembers(ctx, roomKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get users for room %s: %v", roomID, err)
	}

	log.Printf("ROOM (%s) MEMBER in Redis, users: %v", roomID, userIDs)

	// Step 2: 활성 사용자 필터링
	activeUsers := []int{}
	for _, sUserID := range userIDs {
		activeKey := "client:active"
		active, err := r.Client.HGet(ctx, activeKey, sUserID).Result()
		if err == redis.Nil || active != "unique-server-id" {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("failed to check active status for user %s: %v", sUserID, err)
		}

		userID, err := strconv.Atoi(sUserID)
		if err != nil {
			log.Printf("sUserID is not number: %s", sUserID)
			continue
		}

		// 활성 사용자 추가
		activeUsers = append(activeUsers, userID)
	}

	log.Printf("ACTIVE USER in Redis, users: %v", activeUsers)

	return activeUsers, nil
}

// Room의 비활성 사용자 ID 리스트를 반환
func (r *RedisClient) GetInActiveUserIDs(roomID string) ([]int, error) {
	// Step 1: Room의 사용자 ID 리스트 가져오기
	// TODO: MongoDB 시작 시 Redis와 동기화 작업이 필요함
	roomKey := fmt.Sprintf("room:%s", roomID)
	userIDs, err := r.Client.SMembers(ctx, roomKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get users for room %s: %v", roomID, err)
	}

	log.Printf("ROOM (%s) MEMBER in Redis, users: %v", roomID, userIDs)

	// Step 2: 활성 사용자 필터링
	inactiveUsers := []int{}
	for _, sUserID := range userIDs {
		activeKey := "client:active"
		active, err := r.Client.HGet(ctx, activeKey, sUserID).Result()
		if err == redis.Nil && active != "unique-server-id" {
			userID, err := strconv.Atoi(sUserID)
			if err != nil {
				log.Printf("sUserID is not number: %s", sUserID)
				continue
			}

			// 활성 사용자 추가
			inactiveUsers = append(inactiveUsers, userID)
		} else if err != nil {
			return nil, fmt.Errorf("failed to check active status for user %s: %v", sUserID, err)
		}
	}

	log.Printf("INACTIVE USER in Redis, users: %v", inactiveUsers)

	return inactiveUsers, nil
}

// Room 내 존재하는 user의 ID 리스트 반환
func (r *RedisClient) GetRoomUserIDs(roomID string) ([]string, error) {
	// Step 1: Room의 사용자 ID 리스트 가져오기
	roomKey := fmt.Sprintf("room:%s", roomID)
	userIDs, err := r.Client.SMembers(ctx, roomKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get users for room %s: %v", roomID, err)
	}

	return userIDs, nil
}

func (r *RedisClient) JoinRoom(roomID string, userID int) error {
	roomKey := fmt.Sprintf("join_room:%s", roomID)
	err := r.Client.SAdd(ctx, roomKey, strconv.Itoa(userID)).Err()
	if err != nil {
		return fmt.Errorf("failed to join room %s for user %d: %v", roomID, userID, err)
	}
	log.Printf("User %d joined room %s", userID, roomID)
	return nil
}

// TODO: 사용자 웹소켓이 끊기면 Leave해줘야함
func (r *RedisClient) LeaveRoom(roomID string, userID int) error {
	roomKey := fmt.Sprintf("join_room:%s", roomID)
	err := r.Client.SRem(ctx, roomKey, strconv.Itoa(userID)).Err()
	if err != nil {
		return fmt.Errorf("failed to leave room %s for user %d: %v", roomID, userID, err)
	}
	log.Printf("User %d left room %s", userID, roomID)
	return nil
}

func (r *RedisClient) GetJoinedUser(roomID string) ([]int, error) {
	roomKey := fmt.Sprintf("join_room:%s", roomID)
	sUserIDs, err := r.Client.SMembers(ctx, roomKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get users for room %s: %v", roomID, err)
	}

	userIDs := make([]int, 0, len(sUserIDs))
	for _, sUserID := range sUserIDs {
		id, convErr := strconv.Atoi(sUserID)
		if convErr != nil {
			return nil, fmt.Errorf("failed to convert user ID '%s' to int: %v", sUserID, convErr)
		}
		userIDs = append(userIDs, id)
	}

	log.Printf("Users in room %s: %v", roomID, userIDs)
	return userIDs, nil
}

// Room Timeout 사용자 추가
func (r *RedisClient) AddTimeoutUser(roomID string, userID int) error {
	timeoutKey := fmt.Sprintf("room_timeout:%s", roomID)
	err := r.Client.SAdd(ctx, timeoutKey, strconv.Itoa(userID)).Err()
	if err != nil {
		return fmt.Errorf("failed to add timeout user %d to room %s: %v", userID, roomID, err)
	}
	log.Printf("User %d marked as timeout in room %s", userID, roomID)
	return nil
}

// Room Timeout된 사용자 개수 취합
func (r *RedisClient) GetTimeoutUserCount(roomID string) (int64, error) {
	timeoutKey := fmt.Sprintf("room_timeout:%s", roomID)
	count, err := r.Client.SCard(ctx, timeoutKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get timeout user count for room %s: %v", roomID, err)
	}
	return count, nil
}

func (r *RedisClient) ClearRoomTimeout(roomID string) error {
	timeoutKey := fmt.Sprintf("room_timeout:%s", roomID)
	err := r.Client.Del(ctx, timeoutKey).Err()
	if err != nil {
		return fmt.Errorf("failed to clear room timeout data for room %s: %v", roomID, err)
	}
	log.Printf("Cleared timeout data for room %s", roomID)
	return nil
}

func (r *RedisClient) SaveUserChoice(roomID string, userID, selectedUserID int) error {
	choiceKey := fmt.Sprintf("final_choice_room:%s", roomID)
	err := r.Client.HSet(ctx, choiceKey, strconv.Itoa(userID), strconv.Itoa(selectedUserID)).Err()
	if err != nil {
		return fmt.Errorf("failed to save user choice for room %s, user %d: %v", roomID, userID, err)
	}
	log.Printf("User %d selected %d in room %s", userID, selectedUserID, roomID)
	return nil
}

func (r *RedisClient) IsAllChoicesCompleted(roomID string, totalUsers int64) (bool, error) {
	choiceKey := fmt.Sprintf("final_choice_room:%s", roomID)
	choiceCount, err := r.Client.HLen(ctx, choiceKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to get choice count for room %s: %v", roomID, err)
	}

	log.Printf("Room %s: %d/%d users have made their choices", roomID, choiceCount, totalUsers)
	return choiceCount == totalUsers, nil
}

func (r *RedisClient) GetAllChoices(roomID string) (*types.FinalChoiceResultMessage, error) {
	choiceKey := fmt.Sprintf("final_choice_room:%s", roomID)
	choicesMap, err := r.Client.HGetAll(ctx, choiceKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get all choices for room %s: %v", roomID, err)
	}

	// 변환된 데이터 저장
	var choices []types.UserChoice
	for userID, selectedUserID := range choicesMap {
		nUserID, _ := strconv.Atoi(userID)
		nSelectedUserID, _ := strconv.Atoi(selectedUserID)

		choices = append(choices, types.UserChoice{
			UserID:         nUserID,
			SelectedUserID: nSelectedUserID,
		})
	}

	finalChoices := &types.FinalChoiceResultMessage{
		RoomID:  roomID,
		Choices: choices,
	}

	log.Printf("Final choices for room %s: %+v", roomID, finalChoices)
	return finalChoices, nil
}

func (r *RedisClient) ClearFinalChoiceRoom(roomID string) error {
	choiceKey := fmt.Sprintf("final_choice_room:%s", roomID)
	err := r.Client.Del(ctx, choiceKey).Err()
	if err != nil {
		return fmt.Errorf("failed to clear final choice room data for room %s: %v", roomID, err)
	}
	log.Printf("Cleared final choice data for room %s", roomID)
	return nil
}
