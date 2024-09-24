package main

import (
	// Redis 패키지 import

	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/baaami/dorandoran/auth/pkg/data"
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

	// [Network] 카카오 API 호출을 통해 access token 검증
	kakaoResponse, err := GetKaKaoUserInfoByAccessToken(requestData.AccessToken)
	if err != nil {
		http.Error(w, "Invalid Kakao token", http.StatusUnauthorized)
	}

	// 사용자 정보에서 카카오 사용자 ID 추출
	kakaoUserID := fmt.Sprintf("%v", kakaoResponse["id"])

	// [Hub Network] User 서비스에 API를 호출하여 존재하는 회원인지 확인
	existUser, err := GetExistUserByUserSrv(types.KAKAO, kakaoUserID)
	if err != nil {
		fmt.Printf("Error occurred while checking user existence: %v\n", err)
		return
	}

	var sessionID string

	if (existUser == data.User{}) {
		// 유저가 존재하지 않는 경우 -> 회원가입 진행
		newUserID, err := RegisterNewUser(app, kakaoUserID)
		if err != nil {
			http.Error(w, "Failed to register new user", http.StatusInternalServerError)
			return
		}

		// 생성된 user ID로 세션 생성
		sessionID = app.RedisClient.CreateSession(strconv.Itoa(newUserID))

	} else {
		// 유저가 존재하는 경우 -> 세션 존재 여부 확인
		sessionID, err = app.RedisClient.GetSessionByUserID(strconv.Itoa(existUser.ID))
		if err == nil && sessionID != "" {
			// 기존 세션이 존재하는 경우 -> 그대로 사용
		} else {
			// 기존 세션이 존재하지 않는 경우 -> 새 세션 생성
			sessionID = app.RedisClient.CreateSession(strconv.Itoa(existUser.ID))
			if err != nil {
				http.Error(w, "Failed to create session", http.StatusInternalServerError)
				return
			}
		}
	}

	SetSessionCookie(&w, sessionID)

	// 클라이언트에게 로그인 성공 응답
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Login successful, session ID issued"))
}

// [Network] 카카오 API 호출을 통해 access token 검증
func GetKaKaoUserInfoByAccessToken(accessToken string) (map[string]interface{}, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", KAKAO_API_USER_INFO_URL, nil)
	req.Header.Set("Authorization", fmt.Sprintf("BearerO %s", accessToken))

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		log.Printf("Invalid Kakao token, statuscode: %d, err: %s", resp.StatusCode, err.Error())
		return nil, err
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	var kakaoResponse map[string]interface{}
	json.Unmarshal(body, &kakaoResponse)

	return kakaoResponse, nil
}

// [Hub Network] User 서비스에 API를 호출하여 존재하는 회원인지 확인
func GetExistUserByUserSrv(snsType int, snsID string) (data.User, error) {
	client := &http.Client{
		Timeout: time.Second * 10, // 요청 타임아웃 설정
	}

	// 요청 URL 생성
	url := fmt.Sprintf("http://user-service/exist?sns_type=%d&sns_id=%s", snsType, snsID)

	// GET 요청 생성
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return data.User{}, fmt.Errorf("failed to create request: %v", err)
	}

	// 요청 실행
	resp, err := client.Do(req)
	if err != nil {
		return data.User{}, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// 응답 처리
	if resp.StatusCode == http.StatusNotFound {
		// 유저가 존재하지 않는 경우
		return data.User{}, nil
	} else if resp.StatusCode != http.StatusOK {
		// 다른 에러가 발생한 경우
		return data.User{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// 응답 본문에서 유저 정보 디코딩
	var user data.User
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		return data.User{}, fmt.Errorf("failed to decode response: %v", err)
	}

	// 유저가 존재하는 경우
	return user, nil
}

// [Hub Network] User 서비스에 API를 호출하여 새로운 사용자 생성
func RegisterNewUser(app *Config, kakaoUserID string) (int, error) {
	newUser := data.User{
		SnsType: types.KAKAO, // Kakao SNS 유형
		SnsID:   kakaoUserID, // Kakao 사용자 ID
	}

	// user-service로 POST 요청 보내기
	client := &http.Client{}
	reqBody, err := json.Marshal(newUser)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal new user data: %v", err)
	}

	req, err := http.NewRequest("POST", "http://user-service/register", bytes.NewBuffer(reqBody))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send request to user-service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("failed to create user, status code: %d", resp.StatusCode)
	}

	var createdUser data.User
	err = json.NewDecoder(resp.Body).Decode(&createdUser)
	if err != nil {
		return 0, fmt.Errorf("failed to decode response: %v", err)
	}

	return createdUser.ID, nil
}

func SetSessionCookie(w *http.ResponseWriter, sessionID string) {
	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
	}
	http.SetCookie(*w, cookie)
}
