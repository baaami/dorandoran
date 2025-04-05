package redis

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

func (r *RedisClient) JoinRoom(roomID string, userID int) error {
	roomKey := fmt.Sprintf("join_room:%s", roomID)
	err := r.Client.SAdd(ctx, roomKey, strconv.Itoa(userID)).Err()
	if err != nil {
		return fmt.Errorf("failed to join room %s for user %d: %v", roomID, userID, err)
	}
	log.Printf("User %d joined room %s", userID, roomID)
	return nil
}

func (r *RedisClient) LeaveRoom(roomID string, userID int) error {
	roomKey := fmt.Sprintf("join_room:%s", roomID)
	err := r.Client.SRem(ctx, roomKey, strconv.Itoa(userID)).Err()
	if err != nil {
		return fmt.Errorf("❌ Redis LeaveRoom 실패: %w", err)
	}
	log.Printf("🚪 User %d removed from room %s in Redis", userID, roomID)
	return nil
}

// Redis에서 모든 Room ID 가져오기
func (r *RedisClient) GetAllRoomsFromRedis() ([]string, error) {
	ctx := context.Background()

	// Redis에서 방 목록 가져오기
	roomIDs, err := r.Client.SMembers(ctx, "rooms:list").Result()
	if err != nil {
		log.Printf("Failed to get room list from Redis: %v", err)
		return nil, err
	}

	return roomIDs, nil
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

// Redis에서 채팅방 남은 시간 가져오기
func (r *RedisClient) GetRoomRemainingTime(roomID string) (int, error) {
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

func (r *RedisClient) GetRoomStatus(roomID string) (int, error) {
	statusKey := fmt.Sprintf("room_status:%s", roomID)
	statusStr, err := r.Client.Get(ctx, statusKey).Result()
	if err == redis.Nil {
		return 0, fmt.Errorf("status not found for room %s", roomID) // 상태가 없으면 에러 반환
	} else if err != nil {
		return 0, fmt.Errorf("failed to get status for room %s: %v", roomID, err)
	}

	status, convErr := strconv.Atoi(statusStr)
	if convErr != nil {
		return 0, fmt.Errorf("failed to convert status for room %s: %v", roomID, convErr)
	}

	log.Printf("Get status for room %s: %d", roomID, status)
	return status, nil
}

// 밸런스 게임 시작 타이머 설정
func (r *RedisClient) SetBalanceGameTimer(roomID string, duration time.Duration) error {
	ctx := context.Background()

	// 방 Timeout 목록에 추가
	err := r.Client.Set(ctx, fmt.Sprintf("balance_game_timer:%s", roomID), duration.Seconds(), duration).Err()
	if err != nil {
		log.Printf("Failed to set balance game timer for RoomID %s: %v", roomID, err)
		return err
	}

	// 밸런스 게임 타이머 목록에 추가
	err = r.Client.SAdd(ctx, "rooms:balance_game", roomID).Err()
	if err != nil {
		log.Printf("Failed to add RoomID %s to balance game rooms: %v", roomID, err)
		return err
	}

	log.Printf("Balance game timer set for RoomID %s: %v seconds", roomID, duration.Seconds())
	return nil
}

// 밸런스 게임 타이머가 설정된 모든 방 조회
func (r *RedisClient) GetAllBalanceGameRooms() ([]string, error) {
	ctx := context.Background()

	roomIDs, err := r.Client.SMembers(ctx, "rooms:balance_game").Result()
	if err != nil {
		log.Printf("Failed to get balance game rooms from Redis: %v", err)
		return nil, err
	}

	return roomIDs, nil
}

// 밸런스 게임 타이머 남은 시간 조회
func (r *RedisClient) GetBalanceGameRemainingTime(roomID string) (int, error) {
	ctx := context.Background()

	ttl, err := r.Client.TTL(ctx, fmt.Sprintf("balance_game_timer:%s", roomID)).Result()
	if err != nil {
		log.Printf("Failed to get remaining time for balance game in room %s: %v", roomID, err)
		return 0, err
	}

	if ttl <= 0 {
		return 0, nil
	}

	return int(ttl.Seconds()), nil
}

// 밸런스 게임 타이머에서 방 제거
func (r *RedisClient) RemoveBalanceGameRoom(roomID string) error {
	ctx := context.Background()

	// 타이머 키 삭제
	err := r.Client.Del(ctx, fmt.Sprintf("balance_game_timer:%s", roomID)).Err()
	if err != nil {
		log.Printf("Failed to delete balance game timer for room %s: %v", roomID, err)
		return err
	}

	// 방 목록에서 제거
	err = r.Client.SRem(ctx, "rooms:balance_game", roomID).Err()
	if err != nil {
		log.Printf("Failed to remove room %s from balance game rooms: %v", roomID, err)
		return err
	}

	return nil
}

// 밸런스 게임 종료 타이머 설정
func (r *RedisClient) SetBalanceGameFinishTimer(formID string, duration time.Duration) error {
	ctx := context.Background()

	// form Timeout 목록에 추가
	err := r.Client.Set(ctx, fmt.Sprintf("balance_game_finish:%s", formID), duration.Seconds(), duration).Err()
	if err != nil {
		log.Printf("Failed to set balance game finish timer for FormID %s: %v", formID, err)
		return err
	}

	// 밸런스 게임 종료 타이머 목록에 추가
	err = r.Client.SAdd(ctx, "forms:balance_game_finish", formID).Err()
	if err != nil {
		log.Printf("Failed to add FormID %s to balance game finish forms: %v", formID, err)
		return err
	}

	log.Printf("Balance game finish timer set for FormID %s: %v seconds", formID, duration.Seconds())
	return nil
}

// 밸런스 게임 종료 타이머가 설정된 모든 form 조회
func (r *RedisClient) GetAllBalanceGameFinishForms() ([]string, error) {
	ctx := context.Background()

	formIDs, err := r.Client.SMembers(ctx, "forms:balance_game_finish").Result()
	if err != nil {
		log.Printf("Failed to get balance game finish forms from Redis: %v", err)
		return nil, err
	}

	return formIDs, nil
}

// 밸런스 게임 종료 타이머 남은 시간 조회
func (r *RedisClient) GetBalanceGameFinishRemainingTime(formID string) (int, error) {
	ctx := context.Background()

	ttl, err := r.Client.TTL(ctx, fmt.Sprintf("balance_game_finish:%s", formID)).Result()
	if err != nil {
		log.Printf("Failed to get remaining time for balance game finish in form %s: %v", formID, err)
		return 0, err
	}

	if ttl <= 0 {
		return 0, nil
	}

	return int(ttl.Seconds()), nil
}

// 밸런스 게임 종료 타이머에서 form 제거
func (r *RedisClient) RemoveBalanceGameFinishForm(formID string) error {
	ctx := context.Background()

	// 타이머 키 삭제
	err := r.Client.Del(ctx, fmt.Sprintf("balance_game_finish:%s", formID)).Err()
	if err != nil {
		log.Printf("Failed to delete balance game finish timer for form %s: %v", formID, err)
		return err
	}

	// form 목록에서 제거
	err = r.Client.SRem(ctx, "forms:balance_game_finish", formID).Err()
	if err != nil {
		log.Printf("Failed to remove form %s from balance game finish forms: %v", formID, err)
		return err
	}

	return nil
}
