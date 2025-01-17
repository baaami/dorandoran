package main

import (
	// Redis 패키지 import

	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/baaami/dorandoran/auth/pkg/types"
	"github.com/labstack/echo/v4"
)

type Address struct {
	City     string `gorm:"size:100" json:"city"`
	District string `gorm:"size:100" json:"district"`
	Street   string `gorm:"size:100" json:"street"`
}

type User struct {
	ID        int     `gorm:"primaryKey;autoIncrement" json:"id"`
	SnsType   int     `gorm:"index" json:"sns_type"`
	SnsID     string  `gorm:"index" json:"sns_id"`
	Name      string  `gorm:"size:100" json:"name"`
	Gender    int     `json:"gender"`
	Birth     string  `gorm:"size:20" json:"birth"`
	Address   Address `gorm:"embedded;embeddedPrefix:address_" json:"address"`
	GamePoint int     `json:"game_point"`
}

const (
	KAKAO_API_USER_INFO_URL = "https://kapi.kakao.com/v2/user/me"
	NAVER_API_USER_INFO_URL = "https://openapi.naver.com/v1/nid/me"
)

// Kakao 로그인 핸들러
func (app *Config) KakaoLoginHandler(c echo.Context) error {
	var requestData struct {
		AccessToken string `json:"accessToken"`
	}

	if err := c.Bind(&requestData); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	var snsID string

	if strings.HasPrefix(requestData.AccessToken, "masterkey-") {
		parts := strings.Split(requestData.AccessToken, "-")
		if len(parts) == 2 {
			snsID = parts[1]
		} else {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid masterkey format"})
		}
	} else {
		kakaoResponse, err := GetKaKaoUserInfoByAccessToken(requestData.AccessToken)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid Kakao token"})
		}

		idValue, ok := kakaoResponse["id"].(float64)
		if !ok {
			log.Printf("Invalid Kakao Id: %v", kakaoResponse["id"])
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid Kakao Id"})
		}
		snsID = strconv.FormatInt(int64(idValue), 10)
	}

	loginUser, err := GetExistUserByUserSrv(types.KAKAO, snsID)
	if err != nil {
		log.Printf("Error checking user existence: %v\n", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	var sessionID string

	if loginUser == (User{}) {
		loginUser, err = RegisterNewUser(types.KAKAO, snsID)
		if err != nil {
			log.Printf("Failed to register new user")
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to register new user"})
		}
		sessionID = app.RedisClient.CreateSession(loginUser.ID)
	} else {
		sessionID, err = app.RedisClient.GetSessionByUserID(loginUser.ID)
		if err != nil || sessionID == "" {
			sessionID = app.RedisClient.CreateSession(loginUser.ID)
		}
	}

	SetSessionCookie(c, sessionID)

	return c.JSON(http.StatusOK, loginUser)
}

// [Network] 카카오 API 호출을 통해 access token 검증
func GetKaKaoUserInfoByAccessToken(accessToken string) (map[string]interface{}, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", KAKAO_API_USER_INFO_URL, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("Invalid Kakao token, statuscode: %d, err: %s", resp.StatusCode, err.Error())
		return nil, err
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var kakaoResponse map[string]interface{}
	json.Unmarshal(body, &kakaoResponse)

	// kakaoResponse 로그
	log.Printf("Kakao response: %v", kakaoResponse)

	// kakaoResponse에 id key가 있는지 확인
	if _, ok := kakaoResponse["id"]; !ok {
		log.Printf("Kakao response does not contain 'id' field")
		return nil, fmt.Errorf("kakao response does not contain 'id' field")
	}

	return kakaoResponse, nil
}

// Naver 로그인 핸들러
func (app *Config) NaverLoginHandler(c echo.Context) error {
	var requestData struct {
		AccessToken string `json:"accessToken"`
	}

	if err := c.Bind(&requestData); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	var snsID string

	if strings.HasPrefix(requestData.AccessToken, "masterkey-") {
		parts := strings.Split(requestData.AccessToken, "-")
		if len(parts) == 2 {
			snsID = parts[1]
		} else {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid masterkey format"})
		}
	} else {
		naverResponse, err := GetNaverUserInfoByAccessToken(requestData.AccessToken)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid Naver token"})
		}

		responseData, ok := naverResponse["response"].(map[string]interface{})
		if !ok {
			log.Printf("Invalid Naver response: %v", naverResponse)
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid Naver response"})
		}

		idValue, ok := responseData["id"].(string)
		if !ok {
			log.Printf("Invalid Naver Id: %v", responseData["id"])
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid Naver Id"})
		}
		snsID = idValue
	}

	loginUser, err := GetExistUserByUserSrv(types.NAVER, snsID)
	if err != nil {
		log.Printf("Error checking user existence: %v\n", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	var sessionID string

	if loginUser == (User{}) {
		loginUser, err = RegisterNewUser(types.NAVER, snsID)
		if err != nil {
			log.Printf("Failed to register new user")
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to register new user"})
		}
		sessionID = app.RedisClient.CreateSession(loginUser.ID)
	} else {
		sessionID, err = app.RedisClient.GetSessionByUserID(loginUser.ID)
		if err != nil || sessionID == "" {
			sessionID = app.RedisClient.CreateSession(loginUser.ID)
		}
	}

	SetSessionCookie(c, sessionID)

	return c.JSON(http.StatusOK, loginUser)
}

// [Network] 네이버 API 호출을 통해 access token 검증
func GetNaverUserInfoByAccessToken(accessToken string) (map[string]interface{}, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", NAVER_API_USER_INFO_URL, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("Invalid Naver token, status code: %d, err: %v", resp.StatusCode, err)
		return nil, err
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var naverResponse map[string]interface{}
	json.Unmarshal(body, &naverResponse)

	// naverResponse 로그
	log.Printf("Naver response: %v", naverResponse)

	// naverResponse에 response 필드가 있는지 확인
	if _, ok := naverResponse["response"]; !ok {
		log.Printf("Naver response does not contain 'response' field")
		return nil, fmt.Errorf("naver response does not contain 'response' field")
	}

	return naverResponse, nil
}

// [Hub Network] User 서비스에 API를 호출하여 존재하는 회원인지 확인
func GetExistUserByUserSrv(snsType int, snsID string) (User, error) {
	client := &http.Client{
		Timeout: time.Second * 10, // 요청 타임아웃 설정
	}

	// 요청 URL 생성
	url := fmt.Sprintf("http://user-service/exist?sns_type=%d&sns_id=%s", snsType, snsID)

	// GET 요청 생성
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return User{}, fmt.Errorf("failed to create request: %v", err)
	}

	// 요청 실행
	resp, err := client.Do(req)
	if err != nil {
		return User{}, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// 응답 처리
	if resp.StatusCode == http.StatusNotFound {
		// 유저가 존재하지 않는 경우
		return User{}, nil
	} else if resp.StatusCode != http.StatusOK {
		// 다른 에러가 발생한 경우
		return User{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// 응답 본문에서 유저 정보 디코딩
	var user User

	// 응답 본문 로깅 추가
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return User{}, fmt.Errorf("failed to read response body: %v", err)
	}

	log.Printf("Raw response body: %s", string(body))

	// 본문을 다시 디코딩
	err = json.Unmarshal(body, &user)
	if err != nil {
		return User{}, fmt.Errorf("failed to decode response: %v", err)
	}

	// 유저가 존재하는 경우
	return user, nil
}

// [Hub Network] User 서비스에 API를 호출하여 새로운 사용자 생성
func RegisterNewUser(snsType int, snsID string) (User, error) {
	newUser := User{
		SnsType:   snsType, // Kakao SNS 유형
		SnsID:     snsID,   // Kakao 사용자 ID
		GamePoint: types.DEFAULT_GAME_POINT,
	}

	// user-service로 POST 요청 보내기
	client := &http.Client{}
	reqBody, err := json.Marshal(newUser)
	if err != nil {
		return User{}, fmt.Errorf("failed to marshal new user data: %v", err)
	}

	req, err := http.NewRequest("POST", "http://user-service/register", bytes.NewBuffer(reqBody))
	if err != nil {
		return User{}, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return User{}, fmt.Errorf("failed to send request to user-service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return User{}, fmt.Errorf("failed to create user, status code: %d", resp.StatusCode)
	}

	var createdUser User
	err = json.NewDecoder(resp.Body).Decode(&createdUser)
	if err != nil {
		return User{}, fmt.Errorf("failed to decode response: %v", err)
	}

	log.Printf("Registered User: %v", createdUser)

	return createdUser, nil
}

// 세션 쿠키 설정
func SetSessionCookie(c echo.Context, sessionID string) {
	cookie := new(http.Cookie)
	cookie.Name = "session_id"
	cookie.Value = sessionID
	cookie.HttpOnly = true
	cookie.Secure = true
	cookie.Path = "/"
	c.SetCookie(cookie)
}
