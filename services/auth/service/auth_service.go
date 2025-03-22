package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"solo/services/auth/repository"
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

func (s *AuthService) CreateSession(userID int) string {
	return s.repo.CreateSession(userID)
}
