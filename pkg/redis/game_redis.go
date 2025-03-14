package redis

import (
	"context"
	"fmt"
	"log"
	"solo/pkg/types/commontype"
	"solo/pkg/utils/stype"

	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

func (r *RedisClient) RegisterActiveUser(userID int, serverID string) error {
	return r.Client.HSet(ctx, "client:active", strconv.Itoa(userID), serverID).Err()
}

func (r *RedisClient) UnregisterActiveUser(userID int) error {
	return r.Client.HDel(ctx, "client:active", strconv.Itoa(userID)).Err()
}

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
		if err == redis.Nil || active != commontype.DEFAULT_TEMP_SERVER_ID {
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

func (r *RedisClient) AddChatTimeoutUser(roomID string, userID int) error {
	timeoutKey := fmt.Sprintf("chat_timeout:%s", roomID)
	err := r.Client.SAdd(ctx, timeoutKey, strconv.Itoa(userID)).Err()
	if err != nil {
		return fmt.Errorf("failed to add timeout user %d to room %s: %v", userID, roomID, err)
	}
	log.Printf("User %d marked as timeout in room %s", userID, roomID)
	return nil
}

func (r *RedisClient) GetChatTimeoutUserCount(roomID string) (int64, error) {
	timeoutKey := fmt.Sprintf("chat_timeout:%s", roomID)
	count, err := r.Client.SCard(ctx, timeoutKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get timeout user count for room %s: %v", roomID, err)
	}
	return count, nil
}

func (r *RedisClient) ClearChatTimeout(roomID string) error {
	timeoutKey := fmt.Sprintf("chat_timeout:%s", roomID)
	err := r.Client.Del(ctx, timeoutKey).Err()
	if err != nil {
		return fmt.Errorf("failed to clear room timeout data for room %s: %v", roomID, err)
	}
	log.Printf("Cleared timeout data for room %s", roomID)
	return nil
}

func (r *RedisClient) GetAllChoiceRoomsFromRedis() ([]string, error) {
	ctx := context.Background()

	// Redis에서 방 목록 가져오기
	roomIDs, err := r.Client.SMembers(ctx, "rooms:choice").Result()
	if err != nil {
		log.Printf("Failed to get room choice from Redis: %v", err)
		return nil, err
	}

	return roomIDs, nil
}

func (r *RedisClient) GetChoiceRoomRemainingTime(roomID string) (int, error) {
	ctx := context.Background()

	ttl, err := r.Client.TTL(ctx, roomID).Result()
	if err != nil {
		log.Printf("Failed to get remaining time for RoomID %s: %v", roomID, err)
		return 0, err
	}

	if ttl <= 0 {
		log.Printf("RoomID %s has no remaining time or is expired", roomID)
		return 0, nil // 타임아웃이 만료된 경우
	}

	return int(ttl.Seconds()), nil
}

func (r *RedisClient) GetRoomUserIDs(roomID string) ([]string, error) {
	// Step 1: Room의 사용자 ID 리스트 가져오기
	roomKey := fmt.Sprintf("room:%s", roomID)
	userIDs, err := r.Client.SMembers(ctx, roomKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get users for room %s: %v", roomID, err)
	}

	return userIDs, nil
}

func (r *RedisClient) SetFinalChoiceTimeout(roomID string, duration time.Duration) error {
	ctx := context.Background()

	// 방 Timeout 목록에 추가
	err := r.Client.Set(ctx, roomID, duration.Seconds(), duration).Err()
	if err != nil {
		log.Printf("Failed to set room timeout for RoomID %s: %v", roomID, err)
		return err
	}

	// 방 목록에 추가
	err = r.Client.SAdd(ctx, "rooms:choice", roomID).Err()
	if err != nil {
		log.Printf("Failed to add RoomID  d%s to rooms choice: %v", roomID, err)
		return err
	}

	log.Printf("Final Choice timeout set for RoomID %s: %v seconds", roomID, duration.Seconds())
	return nil
}

func (r *RedisClient) RemoveChoiceRoomFromRedis(roomID string) error {
	ctx := context.Background()

	// Redis에서 방 제거
	err := r.Client.Del(ctx, roomID).Err()
	if err != nil {
		log.Printf("Failed to delete room %s from Redis: %v", roomID, err)
		return err
	}

	// 방 목록에서도 제거
	err = r.Client.SRem(ctx, "rooms:choice", roomID).Err()
	if err != nil {
		log.Printf("Failed to remove RoomID %s from rooms choice: %v", roomID, err)
		return err
	}

	log.Printf("RoomID %s removed from Redis", roomID)
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

func (r *RedisClient) GetAllChoices(roomID string) (*stype.FinalChoiceResultMessage, error) {
	choiceKey := fmt.Sprintf("final_choice_room:%s", roomID)
	choicesMap, err := r.Client.HGetAll(ctx, choiceKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get all choices for room %s: %v", roomID, err)
	}

	// 변환된 데이터 저장
	var choices []stype.UserChoice
	for userID, selectedUserID := range choicesMap {
		nUserID, _ := strconv.Atoi(userID)
		nSelectedUserID, _ := strconv.Atoi(selectedUserID)

		choices = append(choices, stype.UserChoice{
			UserID:         nUserID,
			SelectedUserID: nSelectedUserID,
		})
	}

	finalChoices := &stype.FinalChoiceResultMessage{
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
