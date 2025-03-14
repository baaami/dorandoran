package repository

import (
	"encoding/json"
	"fmt"
	"log"
	"solo/pkg/models"
	"solo/pkg/redis"
	"solo/pkg/types/commontype"

	"time"
)

type AuthRepository struct {
	RedisClient *redis.RedisClient
}

func NewAuthRepository(redisClient *redis.RedisClient) *AuthRepository {
	return &AuthRepository{RedisClient: redisClient}
}

// 세션 생성 및 저장
func (repo *AuthRepository) CreateSession(userID int) string {
	sessionID := fmt.Sprintf("session-%d-%d", userID, time.Now().Unix())
	err := repo.RedisClient.Set(sessionID, userID, 24*time.Hour)
	if err != nil {
		log.Printf("Failed to create session: %v", err)
		return ""
	}
	return sessionID
}

// 세션 조회
func (repo *AuthRepository) GetSessionByUserID(userID int) (string, error) {
	sessionKey := fmt.Sprintf("session-%d", userID)
	sessionID, err := repo.RedisClient.Get(sessionKey)
	if err != nil {
		return "", err
	}
	return sessionID, nil
}

// 사용자 조회 또는 생성
func (repo *AuthRepository) FindOrCreateUser(snsType int, snsID string) (models.User, string, error) {
	userKey := fmt.Sprintf("user:%d:%s", snsType, snsID)

	// Redis에서 사용자 조회
	userData, err := repo.RedisClient.Get(userKey)
	if err == nil && userData != "" {
		var user models.User
		err = json.Unmarshal([]byte(userData), &user)
		if err == nil {
			sessionID := repo.CreateSession(user.ID)
			return user, sessionID, nil
		}
	}

	// 사용자가 없으면 새로 등록
	newUser := models.User{
		SnsType:    snsType,
		SnsID:      snsID,
		GameStatus: commontype.USER_STATUS_STANDBY,
		GamePoint:  commontype.DEFAULT_GAME_POINT,
	}

	userJSON, _ := json.Marshal(newUser)
	err = repo.RedisClient.Set(userKey, string(userJSON), 0)
	if err != nil {
		return models.User{}, "", fmt.Errorf("failed to save new user: %v", err)
	}

	sessionID := repo.CreateSession(newUser.ID)
	return newUser, sessionID, nil
}
