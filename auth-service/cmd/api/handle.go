package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

const kakaoTokenURL = "https://kauth.kakao.com/oauth/token"
const kakaoUserInfoURL = "https://kapi.kakao.com/v2/user/me"

type KakaoUserInfo struct {
	ID          int64          `json:"id"`
	ConnectedAt string         `json:"connected_at"`
	Properties  UserProperties `json:"properties"`
	Account     KakaoAccount   `json:"kakao_account"`
}

type UserProperties struct {
	Nickname       string `json:"nickname"`
	ProfileImage   string `json:"profile_image"`
	ThumbnailImage string `json:"thumbnail_image"`
}

type KakaoAccount struct {
	Email    string `json:"email"`
	AgeRange string `json:"age_range"`
	Gender   string `json:"gender"`
}

// KakaoLoginRequest: 카카오 로그인 요청 구조체
type KakaoLoginRequest struct {
	Code string `json:"code"`
}

// KakaoTokenResponse: 카카오에서 발급된 토큰을 저장하는 구조체
type KakaoTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// 카카오 로그인 API
func (app *Config) kakaoLogin(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Auth] Request Receive, %v", r.Method)

	var req KakaoLoginRequest

	// 요청 바디에서 코드 읽기
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Code == "" {
		log.Printf("Failed to NewDecoder, err: %v", err.Error())
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	log.Printf("req.Code: %v", req.Code)

	// 카카오에 access token 요청
	tokenResp, err := requestKakaoAccessToken(req.Code)
	if err != nil {
		log.Printf("Failed to requestKakaoAccessToken, err: %v", err.Error())
		http.Error(w, "Failed to request Kakao access token", http.StatusInternalServerError)
		return
	}

	log.Printf("tokenResp.AccessToken: %v", tokenResp.AccessToken)

	// 토큰을 사용해 카카오 사용자 정보 요청
	userInfo, err := requestKakaoUserInfo(tokenResp.AccessToken)
	if err != nil {
		log.Printf("Failed to requestKakaoUserInfo, err: %v", err.Error())
		http.Error(w, "Failed to retrieve Kakao user info", http.StatusInternalServerError)
		return
	}

	// TODO: RabbitMQ 통해 유저 생성 이벤트 발생

	// 사용자 정보를 응답으로 반환
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(userInfo) // 구조체를 JSON으로 응답
}

// 카카오 Access Token 요청
func requestKakaoAccessToken(code string) (*KakaoTokenResponse, error) {
	client := &http.Client{}

	// 카카오 토큰 요청 파라미터 설정
	data := fmt.Sprintf("grant_type=authorization_code&client_id=%s&redirect_uri=%s&code=%s",
		os.Getenv("KAKAO_APP_KEY"),      // Kakao Client ID
		os.Getenv("KAKAO_REDIRECT_URI"), // Redirect URI
		code)

	req, err := http.NewRequest("POST", kakaoTokenURL, strings.NewReader(data))
	if err != nil {
		return nil, err
	}

	// Content-Type 설정
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 요청 실행
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 응답 데이터 처리
	var tokenResp KakaoTokenResponse
	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	if err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

// 카카오 사용자 정보 요청
func requestKakaoUserInfo(accessToken string) ([]byte, error) {
	client := &http.Client{}

	// 카카오 사용자 정보 요청
	req, err := http.NewRequest("GET", kakaoUserInfoURL, nil)
	if err != nil {
		return nil, err
	}

	// Authorization 헤더에 Bearer 토큰 추가
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// 요청 실행
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 응답 데이터 읽기
	userInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// JSON 데이터를 읽을 수 있는 형태로 변환하여 출력
	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, userInfo, "", "  ")
	if err != nil {
		log.Printf("Failed to format user info JSON: %v", err)
		return nil, err
	}

	// 로그에 JSON 응답 출력
	log.Printf("Kakao User Info: %s", prettyJSON.String())

	return userInfo, nil
}
