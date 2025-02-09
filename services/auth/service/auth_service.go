package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"solo/pkg/types/commontype"
	"solo/services/auth/repository"
	"time"
)

const (
	KAKAO_API_USER_INFO_URL = "https://kapi.kakao.com/v2/user/me"
	NAVER_API_USER_INFO_URL = "https://openapi.naver.com/v1/nid/me"
)

type AuthService struct {
	repo *repository.AuthRepository
}

func NewAuthService(repo *repository.AuthRepository) *AuthService {
	return &AuthService{repo: repo}
}

// Kakao 토큰 검증 및 SNS ID 반환
func (s *AuthService) VerifyKakaoAccessToken(accessToken string) (string, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", KAKAO_API_USER_INFO_URL, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("invalid Kakao token")
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var kakaoResponse map[string]interface{}
	json.Unmarshal(body, &kakaoResponse)

	idValue, ok := kakaoResponse["id"].(float64)
	if !ok {
		return "", fmt.Errorf("invalid Kakao Id")
	}

	return fmt.Sprintf("%d", int64(idValue)), nil
}

// Naver 토큰 검증 및 SNS ID 반환
func (s *AuthService) VerifyNaverAccessToken(accessToken string) (string, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", NAVER_API_USER_INFO_URL, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("invalid Naver token")
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var naverResponse map[string]interface{}
	json.Unmarshal(body, &naverResponse)

	responseData, ok := naverResponse["response"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid Naver response")
	}

	idValue, ok := responseData["id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid Naver Id")
	}

	return idValue, nil
}

// 로그인 및 회원가입 처리
func (s *AuthService) HandleLogin(snsType int, snsID string) (interface{}, string, error) {
	user, sessionID, err := s.repo.FindOrCreateUser(snsType, snsID)
	if err != nil {
		return nil, "", err
	}

	return user, sessionID, nil
}

func (s *AuthService) GetExistUserByUserSrv(snsType int, snsID string) (commontype.User, error) {
	client := &http.Client{
		Timeout: time.Second * 10, // 요청 타임아웃 설정
	}

	url := fmt.Sprintf("%s/exist?sns_type=%d&sns_id=%s", commontype.UserServiceBaseURL, snsType, snsID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return commontype.User{}, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return commontype.User{}, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return commontype.User{}, nil // 유저가 존재하지 않음
	} else if resp.StatusCode != http.StatusOK {
		return commontype.User{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var user commontype.User
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return commontype.User{}, fmt.Errorf("failed to read response body: %v", err)
	}

	err = json.Unmarshal(body, &user)
	if err != nil {
		return commontype.User{}, fmt.Errorf("failed to decode response: %v", err)
	}

	return user, nil
}

func (s *AuthService) RegisterNewUser(snsType int, snsID string) (commontype.User, error) {
	newUser := commontype.User{
		SnsType:    snsType,
		SnsID:      snsID,
		GameStatus: commontype.USER_STATUS_STANDBY,
		GamePoint:  commontype.DEFAULT_GAME_POINT,
	}

	client := &http.Client{}
	reqBody, err := json.Marshal(newUser)
	if err != nil {
		return commontype.User{}, fmt.Errorf("failed to marshal new user data: %v", err)
	}

	url := fmt.Sprintf("%s/register", commontype.UserServiceBaseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return commontype.User{}, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return commontype.User{}, fmt.Errorf("failed to send request to user-service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return commontype.User{}, fmt.Errorf("failed to create user, status code: %d", resp.StatusCode)
	}

	var createdUser commontype.User
	err = json.NewDecoder(resp.Body).Decode(&createdUser)
	if err != nil {
		return commontype.User{}, fmt.Errorf("failed to decode response: %v", err)
	}

	log.Printf("Registered User: %v", createdUser)
	return createdUser, nil
}

func (s *AuthService) CreateSession(userID int) string {
	return s.repo.CreateSession(userID)
}
