package manager

import (
	"context"
	"log"
	"time"

	"github.com/baaami/dorandoran/chat/pkg/data"
	"github.com/baaami/dorandoran/chat/pkg/event"
	"github.com/baaami/dorandoran/chat/pkg/redis"
)

type RoomManager struct {
	RedisClient *redis.RedisClient
	Emitter     *event.Emitter
	Models      *data.Models
}

func NewRoomManager(redisClient *redis.RedisClient, emitter *event.Emitter, models data.Models) *RoomManager {
	return &RoomManager{
		RedisClient: redisClient,
		Emitter:     emitter,
		Models:      &models,
	}
}

// Redis에 채팅방 타임아웃 설정
func (rm *RoomManager) SetRoomTimeout(roomID string, duration time.Duration) error {
	ctx := context.Background()

	// 방 Timeout 목록에 추가
	err := rm.RedisClient.Client.Set(ctx, roomID, duration.Seconds(), duration).Err()
	if err != nil {
		log.Printf("Failed to set room timeout for RoomID %s: %v", roomID, err)
		return err
	}

	// 방 목록에 추가
	err = rm.RedisClient.Client.SAdd(ctx, "rooms:list", roomID).Err()
	if err != nil {
		log.Printf("Failed to add RoomID %s to rooms list: %v", roomID, err)
		return err
	}

	log.Printf("Room timeout set for RoomID %s: %v seconds", roomID, duration.Seconds())
	return nil
}

// Redis에서 채팅방 남은 시간 가져오기
func (rm *RoomManager) GetRoomRemainingTime(roomID string) (int, error) {
	ctx := context.Background()

	ttl, err := rm.RedisClient.Client.TTL(ctx, roomID).Result()
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

// 방이 만료되거나 삭제될 때 Redis에서 제거
func (rm *RoomManager) RemoveRoomFromRedis(roomID string) error {
	ctx := context.Background()

	// Redis에서 방 제거
	err := rm.RedisClient.Client.Del(ctx, roomID).Err()
	if err != nil {
		log.Printf("Failed to delete room %s from Redis: %v", roomID, err)
		return err
	}

	// 방 목록에서도 제거
	err = rm.RedisClient.Client.SRem(ctx, "rooms:list", roomID).Err()
	if err != nil {
		log.Printf("Failed to remove RoomID %s from rooms list: %v", roomID, err)
		return err
	}

	log.Printf("RoomID %s removed from Redis", roomID)
	return nil
}

// Redis에서 모든 Room ID 가져오기
func (rm *RoomManager) GetAllRoomsFromRedis() ([]string, error) {
	ctx := context.Background()

	// Redis에서 방 목록 가져오기
	roomIDs, err := rm.RedisClient.Client.SMembers(ctx, "rooms:list").Result()
	if err != nil {
		log.Printf("Failed to get room list from Redis: %v", err)
		return nil, err
	}

	return roomIDs, nil
}

func (rm *RoomManager) PushRoomTimeout(roomID string) error {
	inactiveUsers, err := rm.RedisClient.GetInActiveUserIDs(roomID)
	if err != nil {
		log.Printf("Failed to get inactive users, err: %s", err.Error())
		return err
	}

	event := event.RoomTimeoutEvent{
		RoomID:          roomID,
		InactiveUserIds: inactiveUsers,
	}

	err = rm.Emitter.PushRoomTimeout(event)
	if err != nil {
		log.Printf("Failed to push timeout event for RoomID %s: %v", roomID, err)
		return err
	}

	log.Printf("Timeout event pushed for RoomID: %s", roomID)
	return nil
}

func (rm *RoomManager) MonitorRoomTimeouts() {
	ticker := time.NewTicker(1 * time.Second) // 최대 1초 내에 이벤트 감지
	defer ticker.Stop()

	for range ticker.C {
		// Redis에 저장된 모든 방 ID 가져오기
		rooms, err := rm.GetAllRoomsFromRedis()
		if err != nil {
			log.Printf("Failed to fetch rooms for timeout monitoring: %v", err)
			continue
		}

		for _, roomID := range rooms {
			// 남은 시간이 0 이하인지 확인
			remainingTime, err := rm.GetRoomRemainingTime(roomID)
			if err != nil || remainingTime > 0 {
				continue // 아직 만료되지 않은 방은 스킵
			}

			// 만료된 방에 대해 timeout 이벤트 발행
			err = rm.PushRoomTimeout(roomID)
			if err != nil {
				log.Printf("Failed to handle timeout for RoomID %s: %v", roomID, err)
			}

			// TODO: Redis에서 최종 선택 완료 시 방 삭제
			// err = rm.RemoveRoomFromRedis(roomID)
			// if err != nil {
			// 	log.Printf("Failed to remove expired room %s from Redis: %v", roomID, err)
			// }
		}
	}
}
