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

type User struct {
	ID       int    `gorm:"primaryKey;autoIncrement" json:"id"`
	SnsType  int    `gorm:"index" json:"sns_type"`
	SnsID    int64  `gorm:"index" json:"sns_id"`
	Name     string `gorm:"size:100" json:"name"`
	Nickname string `gorm:"size:100" json:"nickname"`
	Gender   int    `json:"gender"`
	Age      int    `json:"age"`
	Email    string `gorm:"size:100" json:"email"`
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
	return webSocketMsg
}

func TestMatchWebSocketAPI(t *testing.T) {
	participantCount := 10
	sessionIDs := make([]string, participantCount)
	userIDs := make([]string, participantCount)
	conns := make([]*websocket.Conn, participantCount)

	// 채널을 사용하여 응답을 수집
	responseChan := make(chan WebSocketMessage, participantCount)

	// 1. 5명의 참가자가 로그인하여 세션 ID와 유저 ID 발급
	for i := 0; i < participantCount; i++ {
		accessToken := fmt.Sprintf("masterkey-%d", i+1)
		sessionID, userID, err := loginAndGetSessionIDAndUserID(accessToken)
		assert.NoError(t, err)
		sessionIDs[i] = sessionID
		userIDs[i] = userID
	}

	// 2. 5명의 참가자가 WebSocket으로 접속
	for i := 0; i < participantCount; i++ {
		conn, err := connectWebSocket(t, sessionIDs[i], userIDs[i])
		assert.NoError(t, err)
		conns[i] = conn
	}

	// 3. 각 클라이언트가 매칭 요청을 보내고 비동기적으로 응답 대기
	for i := 0; i < participantCount; i++ {
		go func(i int) {
			// 매칭 요청 보내기
			// sendMatchRequest(t, conns[i], userIDs[i])

			// 매칭 응답 수신
			webSocketMsg := receiveMatchResponse(t, conns[i])

			// 응답을 채널에 전달
			responseChan <- webSocketMsg
		}(i)
	}

	// 4. 응답을 수신
	matchResponses := make([]WebSocketMessage, participantCount)
	for i := 0; i < participantCount; i++ {
		matchResponses[i] = <-responseChan

		if matchResponses[i].Type == "match" && matchResponses[i].Status == "success" {
			var matchResp MatchResponse
			err := json.Unmarshal(matchResponses[i].Payload, &matchResp)
			assert.NoError(t, err)
			if matchResp.RoomID != "" {
				t.Logf("%s User allocated room %s", userIDs[i], matchResp.RoomID)
			}
		}
	}

	// 6. WebSocket 연결 닫기
	for _, conn := range conns {
		conn.Close()
	}
}
