package main

import (
	// Redis 패키지 import
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Redis 클라이언트 생성

const KAKAO_API_USER_INFO_URL = "https://kapi.kakao.com/v2/user/me"

// 클라이언트로부터 받은 access token을 검증하는 함수
func (app *Config) KakaoLoginHandler(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		AccessToken string `json:"accessToken"`
	}

	// 클라이언트로부터 받은 access token 파싱
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// 카카오 API 호출을 통해 access token 검증
	client := &http.Client{}
	req, _ := http.NewRequest("GET", KAKAO_API_USER_INFO_URL, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", requestData.AccessToken))

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		http.Error(w, "Invalid Kakao token", http.StatusUnauthorized)
		return
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	var kakaoResponse map[string]interface{}
	json.Unmarshal(body, &kakaoResponse)

	// 사용자 정보에서 카카오 사용자 ID 추출
	kakaoUserID := fmt.Sprintf("%v", kakaoResponse["id"])

	// 기존 세션이 존재하는지 확인
	sessionID, err := app.RedisClient.GetSessionByUserID(kakaoUserID)
	if err != nil {
		// 세션이 존재하지 않으면 새로 생성
		sessionID = app.CreateSession(kakaoUserID)
	}

	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
	}
	http.SetCookie(w, cookie)

	// 클라이언트에게 로그인 성공 응답
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Login successful, session ID issued"))
}

// 세션 생성 함수 (Redis에 세션 저장)
func (app *Config) CreateSession(kakaoUserID string) string {
	// 고유한 세션 ID 생성 (UUID 사용)
	sessionID := uuid.New().String()

	// 세션 만료 시간 설정 (예: 24시간)
	expiresAt := time.Hour * 24

	// Redis에 세션 저장
	err := app.RedisClient.SetSession(sessionID, kakaoUserID, expiresAt)
	if err != nil {
		fmt.Printf("Failed to store session in Redis: %v", err)
	}

	// 생성된 세션 ID 반환
	return sessionID
}
