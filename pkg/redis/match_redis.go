package redis

import (
	"encoding/json"
	"fmt"
	"log"
	"solo/pkg/types/commontype"
	"time"

	"github.com/go-redis/redis/v8"
)

func (r *RedisClient) MonitorAndMatchUsers(coupleCount int) ([]commontype.WaitingUser, error) {
	maleKey := fmt.Sprintf("matching_queue:%d:%d", commontype.MALE, coupleCount)
	femaleKey := fmt.Sprintf("matching_queue:%d:%d", commontype.FEMALE, coupleCount)

	// 1. 남성 대기열에서 가장 오래 기다린 사용자 coupleCount 명 찾기
	oldestMales, err := r.Client.ZRange(ctx, maleKey, 0, int64(coupleCount-1)).Result()
	if err != nil {
		log.Printf("❌ Failed to retrieve male users: %v", err)
		return nil, err
	}
	if len(oldestMales) < coupleCount {
		return nil, nil
	}

	var maleWaitingQueues []commontype.WaitingUser
	for _, maleData := range oldestMales {
		var mq commontype.WaitingUser
		if err := json.Unmarshal([]byte(maleData), &mq); err != nil {
			log.Printf("❌ Failed to unmarshal male user data: %v", err)
			return nil, err
		}
		maleWaitingQueues = append(maleWaitingQueues, mq)
	}

	// 2. 남성 사용자들의 평균 나이를 기준으로 나이 범위 설정
	var totalAge int
	for _, mq := range maleWaitingQueues {
		totalAge += calculateAge(mq.Birth)
	}
	avgAge := float64(totalAge) / float64(len(maleWaitingQueues))
	minAge := avgAge - 10
	maxAge := avgAge + 10

	log.Printf("🔍 Matching males avg age: %.2f, searching females between %.2f and %.2f", avgAge, minAge, maxAge)

	// 3. 나이 범위 내의 여성 상대 찾기
	females, err := r.Client.ZRangeByScore(ctx, femaleKey, &redis.ZRangeBy{
		Min:   fmt.Sprintf("%f", minAge),
		Max:   fmt.Sprintf("%f", maxAge),
		Count: int64(coupleCount),
	}).Result()
	if err != nil {
		log.Printf("❌ Failed to retrieve female users: %v", err)
		return nil, err
	}

	if len(females) < coupleCount {
		log.Printf("ℹ️ Not enough female users for matching. Required: %d, Found: %d", coupleCount, len(females))
		return nil, nil
	}

	// 4. 매칭된 사용자들 처리
	var matchedUsers []commontype.WaitingUser

	// 남성 사용자 파싱
	for _, maleData := range oldestMales {
		var user commontype.WaitingUser
		if err := json.Unmarshal([]byte(maleData), &user); err != nil {
			log.Printf("❌ Failed to unmarshal male user: %v", err)
			continue
		}
		matchedUsers = append(matchedUsers, user)
	}

	// 여성 사용자 파싱
	for _, femaleData := range females {
		var user commontype.WaitingUser
		if err := json.Unmarshal([]byte(femaleData), &user); err != nil {
			log.Printf("❌ Failed to unmarshal female user: %v", err)
			continue
		}
		matchedUsers = append(matchedUsers, user)
	}

	// 매칭된 사용자들을 큐에서 제거
	for _, userData := range oldestMales {
		if err := r.Client.ZRem(ctx, maleKey, userData).Err(); err != nil {
			log.Printf("❌ Failed to remove matched male user from queue: %v", err)
		}
	}
	for _, userData := range females {
		if err := r.Client.ZRem(ctx, femaleKey, userData).Err(); err != nil {
			log.Printf("❌ Failed to remove matched female user from queue: %v", err)
		}
	}

	log.Printf("✅ Successfully matched %d couples", coupleCount)
	return matchedUsers, nil
}

func (r *RedisClient) AddUserToMatchQueue(user commontype.WaitingUser) error {
	age := calculateAge(user.Birth)
	score := float64(age)

	// Redis Sorted Set에 추가
	key := fmt.Sprintf("matching_queue:%d:%d", user.Gender, user.CoupleCount)
	member, _ := json.Marshal(user)

	return r.Client.ZAdd(ctx, key, &redis.Z{
		Score:  score,
		Member: member,
	}).Err()
}

func (r *RedisClient) RemoveUserFromQueue(user commontype.WaitingUser) error {
	genderQueuePrefix := fmt.Sprintf("matching_queue:%d", user.Gender)

	// Serialize WaitingUser to JSON
	userData, err := json.Marshal(user)
	if err != nil {
		log.Printf("Failed to serialize WaitingUser for user %d: %v", user.ID, err)
		return err
	}

	// Iterate over all possible couple counts (2 to 6)
	for coupleCount := commontype.MATCH_COUNT_MIN; coupleCount <= commontype.MATCH_COUNT_MAX; coupleCount++ {
		queueKey := fmt.Sprintf("%s:%d", genderQueuePrefix, coupleCount)

		// Attempt to remove the user from the current queue
		if err := r.Client.LRem(ctx, queueKey, 1, userData).Err(); err != nil {
			log.Printf("Failed to remove user %d from queue %s: %v", user.ID, queueKey, err)
			continue
		}

		// Check if the user was successfully removed
		length, err := r.Client.ZRem(ctx, queueKey, userData).Result()
		if err == nil && length > 0 {
			log.Printf("User %d successfully removed from queue %s", user.ID, queueKey)
			return nil
		}
	}

	return nil
}

func (r *RedisClient) IsUserInMatchQueue(user commontype.WaitingUser) (bool, string, error) {
	genderQueuePrefix := fmt.Sprintf("matching_queue:%d", user.Gender)

	// Serialize WaitingUser to JSON
	userData, err := json.Marshal(user)
	if err != nil {
		log.Printf("Failed to serialize WaitingUser for user %d: %v", user.ID, err)
		return false, "", err
	}

	// Iterate over all possible couple counts (2 to 6)
	for coupleCount := commontype.MATCH_COUNT_MIN; coupleCount <= commontype.MATCH_COUNT_MAX; coupleCount++ {
		queueKey := fmt.Sprintf("%s:%d", genderQueuePrefix, coupleCount)

		// Check if the user exists in the current queue
		exists, err := r.Client.LPos(ctx, queueKey, string(userData), redis.LPosArgs{}).Result()
		if err == nil && exists >= 0 {
			log.Printf("User %d found in queue %s", user.ID, queueKey)
			return true, queueKey, nil
		}
	}

	return false, "", nil
}

func calculateAge(birth string) int {
	// "19960123" 형식을 "1996-01-23" 형식으로 변환
	if len(birth) != 8 {
		log.Printf("Invalid birth date format: %s", birth)
		return 0
	}

	birthStr := fmt.Sprintf("%s-%s-%s",
		birth[:4],  // year
		birth[4:6], // month
		birth[6:8], // day
	)

	birthDate, err := time.Parse("2006-01-02", birthStr)
	if err != nil {
		log.Printf("Failed to parse birth date: %v", err)
		return 0
	}

	age := time.Now().Year() - birthDate.Year()

	// 생일이 아직 지나지 않았다면 나이에서 1을 뺌
	if time.Now().Month() < birthDate.Month() ||
		(time.Now().Month() == birthDate.Month() && time.Now().Day() < birthDate.Day()) {
		age--
	}

	return age
}

func getGenderString(gender int) string {
	if gender == 0 {
		return "male"
	}
	return "female"
}
