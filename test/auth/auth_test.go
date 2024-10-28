package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Address struct {
	City     string `gorm:"size:100" json:"city"`
	District string `gorm:"size:100" json:"district"`
	Street   string `gorm:"size:100" json:"street"`
}

type User struct {
	ID      int     `gorm:"primaryKey;autoIncrement" json:"id"`
	SnsType int     `gorm:"index" json:"sns_type"`
	SnsID   string  `gorm:"index" json:"sns_id"`
	Name    string  `gorm:"size:100" json:"name"`
	Gender  int     `json:"gender"`
	Birth   string  `gorm:"size:20" json:"birth"`
	Address Address `gorm:"embedded;embeddedPrefix:address_" json:"address"`
}

const API_GATEWAY_URL = "http://localhost:2719"

// 로그인 API를 호출하여 세션 ID를 발급받는 함수
func loginAndGetSessionID() (string, error) {
	// 로그인 요청 데이터 설정 (필요한 데이터로 수정)
	loginData := map[string]string{
		"accessToken": "masterkey-1",
	}
	reqBody, _ := json.Marshal(loginData)

	// 로그인 API 호출
	resp, err := http.Post(API_GATEWAY_URL+"/auth/kakao", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 세션 ID가 담긴 쿠키 가져오기
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "session_id" {
			return cookie.Value, nil
		}
	}

	return "", fmt.Errorf("no session ID found")
}

// 테스트 요청을 쿠키와 함께 보내는 함수
func GetUserExist(t *testing.T, sessionID string) {
	// 테스트 요청 생성
	req, err := http.NewRequest(http.MethodGet, API_GATEWAY_URL+"/user/exist?sns_type=0&sns_id=1", nil)
	assert.NoError(t, err)

	// 세션 ID 쿠키 설정
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: sessionID,
	})

	// 클라이언트로 요청 보내기
	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// 응답 상태 코드 확인
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 응답 본문 확인 (필요 시 추가)
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	t.Logf("Response: %s", string(body))
}

// 테스트 요청을 쿠키와 함께 보내는 함수
func UpdateUserProfile(t *testing.T, sessionID string) {

	// 테스트 요청 생성
	req, err := http.NewRequest(http.MethodPut, API_GATEWAY_URL+"/user/update", nil)
	assert.NoError(t, err)

	// 세션 ID 쿠키 설정
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: sessionID,
	})

	// 클라이언트로 요청 보내기
	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// 응답 상태 코드 확인
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 응답 본문 확인 (필요 시 추가)
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	t.Logf("Response: %s", string(body))
}

// 테스트 요청을 쿠키와 함께 보내는 함수
func DeleteUser(t *testing.T, sessionID string) {
	// 테스트 요청 생성
	req, err := http.NewRequest(http.MethodDelete, API_GATEWAY_URL+"/user/delete", nil)
	assert.NoError(t, err)

	// 세션 ID 쿠키 설정
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: sessionID,
	})

	// 클라이언트로 요청 보내기
	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// 응답 상태 코드 확인
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestWithLoginSession(t *testing.T) {
	// 1. 로그인 후 세션 ID 발급받기
	sessionID, err := loginAndGetSessionID()
	assert.NoError(t, err)

	// 2. 발급받은 세션 ID로 요청 보내기
	GetUserExist(t, sessionID)

	// 3. User 삭제하기
	DeleteUser(t, sessionID)
}
