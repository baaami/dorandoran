package redis

import (
	"encoding/json"
	"fmt"
	"log"
	"solo/pkg/types/commontype"

	"github.com/go-redis/redis/v8"
)

func (r *RedisClient) AddUserToMatchQueue(user commontype.WaitingUser) error {
	queueKey := fmt.Sprintf("matching_queue_%s_%d", getGenderString(user.Gender), user.CoupleCount)

	userData, err := json.Marshal(user)
	if err != nil {
		log.Printf("Failed to serialize WaitingUser for user %d: %v", user.ID, err)
		return err
	}

	if err := r.Client.RPush(ctx, queueKey, userData).Err(); err != nil {
		log.Printf("Failed to add user %d to Redis queue %s: %v", user.ID, queueKey, err)
		return err
	}
	log.Printf("User %d added to Redis queue %s", user.ID, queueKey)
	return nil
}

func (r *RedisClient) RemoveUserFromQueue(user commontype.WaitingUser) error {
	genderQueuePrefix := fmt.Sprintf("matching_queue_%s", getGenderString(user.Gender))

	// Serialize WaitingUser to JSON
	userData, err := json.Marshal(user)
	if err != nil {
		log.Printf("Failed to serialize WaitingUser for user %d: %v", user.ID, err)
		return err
	}

	// Iterate over all possible couple counts (2 to 6)
	for coupleCount := commontype.MATCH_COUNT_MIN; coupleCount <= commontype.MATCH_COUNT_MAX; coupleCount++ {
		queueKey := fmt.Sprintf("%s_%d", genderQueuePrefix, coupleCount)

		// Attempt to remove the user from the current queue
		if err := r.Client.LRem(ctx, queueKey, 1, userData).Err(); err != nil {
			log.Printf("Failed to remove user %d from queue %s: %v", user.ID, queueKey, err)
			continue
		}

		// Check if the user was successfully removed
		length, err := r.Client.LLen(ctx, queueKey).Result()
		if err == nil && length > 0 {
			log.Printf("User %d successfully removed from queue %s", user.ID, queueKey)
			return nil
		}
	}

	return nil
}

func (r *RedisClient) IsUserInMatchQueue(user commontype.WaitingUser) (bool, string, error) {
	genderQueuePrefix := fmt.Sprintf("matching_queue_%s", getGenderString(user.Gender))

	// Serialize WaitingUser to JSON
	userData, err := json.Marshal(user)
	if err != nil {
		log.Printf("Failed to serialize WaitingUser for user %d: %v", user.ID, err)
		return false, "", err
	}

	// Iterate over all possible couple counts (2 to 6)
	for coupleCount := commontype.MATCH_COUNT_MIN; coupleCount <= commontype.MATCH_COUNT_MAX; coupleCount++ {
		queueKey := fmt.Sprintf("%s_%d", genderQueuePrefix, coupleCount)

		// Check if the user exists in the current queue
		exists, err := r.Client.LPos(ctx, queueKey, string(userData), redis.LPosArgs{}).Result()
		if err == nil && exists >= 0 {
			log.Printf("User %d found in queue %s", user.ID, queueKey)
			return true, queueKey, nil
		}
	}

	return false, "", nil
}

func getGenderString(gender int) string {
	if gender == 0 {
		return "male"
	}
	return "female"
}
