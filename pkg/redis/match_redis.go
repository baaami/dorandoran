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

	oldestMales, err := r.Client.ZRange(ctx, maleKey, 0, int64(coupleCount-1)).Result()
	if err != nil || len(oldestMales) < coupleCount {
		return nil, err
	}

	maleWaitingQueues := parseWaitingUsers(oldestMales)
	avgAge := calculateAverageAge(maleWaitingQueues)

	females, err := r.findFemalesWithExpandingAgeRange(femaleKey, avgAge, coupleCount)
	if err != nil || len(females) < coupleCount {
		return nil, err
	}

	matchedUsers := append(parseWaitingUsers(oldestMales), parseWaitingUsers(females)...)

	removeMatchedUsersFromQueue(r, maleKey, oldestMales)
	removeMatchedUsersFromQueue(r, femaleKey, females)

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
		if err := r.Client.ZRem(ctx, queueKey, 1, userData).Err(); err != nil {
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

// 연령대 범위를 점진적으로 확장하며 여성 사용자 찾기
func (r *RedisClient) findFemalesWithExpandingAgeRange(femaleKey string, avgAge float64, coupleCount int) ([]string, error) {
	ageRange := 5
	maxAgeRange := 15

	for ageRange <= maxAgeRange {
		minAge := avgAge - float64(ageRange)
		maxAge := avgAge + float64(ageRange)

		females, err := r.Client.ZRangeByScore(ctx, femaleKey, &redis.ZRangeBy{
			Min:   fmt.Sprintf("%f", minAge),
			Max:   fmt.Sprintf("%f", maxAge),
			Count: int64(coupleCount),
		}).Result()
		if err != nil {
			return nil, err
		}

		if len(females) >= coupleCount {
			return females, nil
		}

		ageRange += 5
	}

	return nil, nil
}

// 평균 나이 계산 함수
func calculateAverageAge(users []commontype.WaitingUser) float64 {
	var totalAge int
	for _, user := range users {
		totalAge += calculateAge(user.Birth)
	}
	return float64(totalAge) / float64(len(users))
}

// 사용자 데이터 파싱 함수
func parseWaitingUsers(data []string) []commontype.WaitingUser {
	var users []commontype.WaitingUser
	for _, userData := range data {
		var user commontype.WaitingUser
		if err := json.Unmarshal([]byte(userData), &user); err == nil {
			users = append(users, user)
		}
	}
	return users
}

// 매칭된 사용자 큐에서 제거 함수
func removeMatchedUsersFromQueue(r *RedisClient, key string, users []string) {
	for _, userData := range users {
		if err := r.Client.ZRem(ctx, key, userData).Err(); err != nil {
			log.Printf("❌ Failed to remove matched user from queue: %v", err)
		}
	}
}
