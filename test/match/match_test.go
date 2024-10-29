package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

const (
	API_GATEWAY_URL = "http://localhost:2719"
	WS_MATCH_URL    = "ws://localhost:2719/ws/match"
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

// WebSocketMessage 구조체 정의
type WebSocketMessage struct {
	Type    string          `json:"type"`
	Status  string          `json:"status"`
	Payload json.RawMessage `json:"payload"`
}

// MatchResponse 구조체 정의
type MatchResponse struct {
	RoomID string `json:"room_id"`
}

type UnRegisterMessage struct {
	UserID string `json:"user_id"`
}

// 로그인 API를 호출하여 세션 ID와 유저 ID를 발급받는 함수
func loginAndGetSessionIDAndUserID(accessToken string) (string, string, error) {
	// 로그인 요청 데이터 설정
	loginData := map[string]string{
		"accessToken": accessToken,
	}
	reqBody, err := json.Marshal(loginData)
	if err != nil {
		log.Printf("[ERROR] Error marshaling login data: %v", err)
		return "", "", err
	}

	// 로그인 API 호출
	resp, err := http.Post(API_GATEWAY_URL+"/auth/kakao", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Printf("[ERROR] Error sending login request: %v", err)
		return "", "", err
	}
	defer resp.Body.Close()

	// 상태 코드 확인
	if resp.StatusCode != http.StatusOK {
		log.Printf("[ERROR] Login request failed, status code: %d", resp.StatusCode)
		return "", "", fmt.Errorf("login failed with status code: %d", resp.StatusCode)
	}

	// 세션 ID 추출
	sessionID := ""
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "session_id" {
			sessionID = cookie.Value
			break
		}
	}

	if sessionID == "" {
		log.Printf("[ERROR] No session_id found in cookies")
		return "", "", fmt.Errorf("session_id not found in response cookies")
	}

	// 유저 ID 추출
	var user User
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		log.Printf("[ERROR] Error decoding user data: %v", err)
		return "", "", err
	}

	if user.ID == 0 {
		log.Printf("[ERROR] User ID is 0, failed to retrieve user data: %v", user)
		return "", "", fmt.Errorf("failed to retrieve user ID from response")
	}

	log.Printf("[INFO] User logged in successfully: %v", user)

	// 유저 ID를 문자열로 변환
	userID := strconv.Itoa(user.ID)

	return sessionID, userID, nil
}

func updateProfile(t *testing.T, sessionID string, userID string, gender int) error {
	// Matching 필터 획득
	client := http.Client{}

	updatedUser := User{
		Gender: gender,
	}

	reqBody, err := json.Marshal(updatedUser)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPatch, API_GATEWAY_URL+"/user/update", bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}

	header := http.Header{}

	header.Set("Content-Type", "application/json")
	header.Add("Cookie", "session_id="+sessionID)
	req.Header = header

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to user-service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create user, status code: %d", resp.StatusCode)
	}

	return nil
}

// WebSocket 서버에 접속하는 함수
func connectWebSocket(t *testing.T, sessionID string, userID string) (*websocket.Conn, error) {
	header := http.Header{}
	header.Add("Cookie", "session_id="+sessionID)
	header.Add("X-User-ID", userID)

	conn, _, err := websocket.DefaultDialer.Dial(WS_MATCH_URL, header)
	if err != nil {
		log.Printf("[ERROR] Failed to connect to WebSocket for User %s: %v", userID, err)
		return nil, err
	}
	log.Printf("[INFO] User %s connected to WebSocket", userID)
	return conn, nil
}

// 매칭 결과를 수신하는 함수
func receiveMatchResponse(t *testing.T, conn *websocket.Conn) WebSocketMessage {
	var webSocketMsg WebSocketMessage
	err := conn.ReadJSON(&webSocketMsg)
	if err != nil {
		t.Fatalf("Failed to receive match response: %v", err)
	}

	// 매칭 실패 메시지인 경우 로깅 및 출력
	if webSocketMsg.Status == "fail" {
		t.Logf("Received match failure message: %v", webSocketMsg)
	}

	return webSocketMsg
}

func TestMatchWebSocketAPI(t *testing.T) {
	participantCount := 10
	sessionIDs := make([]string, participantCount)
	userIDs := make([]string, participantCount)
	conns := make([]*websocket.Conn, participantCount)

	// 채널을 사용하여 응답을 수집
	responseChan := make(chan WebSocketMessage, participantCount)

	// 1. 8명의 참가자가 로그인하여 세션 ID와 유저 ID 발급
	for i := 0; i < participantCount; i++ {
		accessToken := fmt.Sprintf("masterkey-%d", i+1)
		sessionID, userID, err := loginAndGetSessionIDAndUserID(accessToken)
		assert.NoError(t, err)
		sessionIDs[i] = sessionID
		userIDs[i] = userID
	}

	// 2. 8명의 참가자가 사용자 프로필 정보 입력
	for i := 0; i < participantCount; i++ {
		// 남성, 여성을 반반으로 변경
		err := updateProfile(t, sessionIDs[i], userIDs[i], i%2)
		if err != nil {
			fmt.Printf("Fail to updateProfile, user: %s, err: %s\n", userIDs[i], err)
			continue
		}
	}

	// 3. 8명의 참가자가 WebSocket으로 접속
	for i := 0; i < participantCount; i++ {
		conn, err := connectWebSocket(t, sessionIDs[i], userIDs[i])
		assert.NoError(t, err)
		conns[i] = conn
	}

	// 4. 각 클라이언트가 매칭 요청을 보내고 비동기적으로 응답 대기
	for i := 0; i < participantCount; i++ {
		go func(i int) {
			webSocketMsg := receiveMatchResponse(t, conns[i])
			responseChan <- webSocketMsg
		}(i)
	}

	// 5. 응답을 수신하고, 성공 또는 실패 메시지 확인
	matchResponses := make([]WebSocketMessage, participantCount)
	for i := 0; i < participantCount; i++ {
		matchResponses[i] = <-responseChan

		if matchResponses[i].Status == "success" {
			var matchResp MatchResponse
			err := json.Unmarshal(matchResponses[i].Payload, &matchResp)
			assert.NoError(t, err)
			if matchResp.RoomID != "" {
				t.Logf("%s User allocated room %s", userIDs[i], matchResp.RoomID)
			}
		} else if matchResponses[i].Status == "fail" {
			t.Logf("%s User match failed due to timeout", userIDs[i])
		}
	}

	// 6. WebSocket 연결 닫기
	for _, conn := range conns {
		conn.Close()
	}
}
