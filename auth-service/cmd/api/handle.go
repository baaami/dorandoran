package main

import (
	// Redis 패키지 import
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/baaami/dorandoran/auth/pkg/types"
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
	snsID := app.RedisClient.MakeSnsID(types.KAKAO, kakaoUserID)

	// 기존 세션이 존재하는지 확인
	sessionID, err := app.RedisClient.GetSessionBySnsID(snsID)
	if err != nil {
		// 세션이 존재하지 않으면 새로 생성
		sessionID = app.RedisClient.CreateSession(snsID)
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
