package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"testing"
	"time"

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
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// MatchMessage 구조체 정의
type MatchMessage struct {
	UserID string `json:"user_id"`
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

// 매칭 요청을 보내고 응답을 기다리는 함수
func sendMatchRequest(t *testing.T, conn *websocket.Conn, userID string) {
	matchMsg := WebSocketMessage{
		Type:    "match",
		Payload: MatchMessage{UserID: userID},
	}

	err := conn.WriteJSON(matchMsg)
	if err != nil {
		t.Fatalf("Failed to send match request for user %s: %v", userID, err)
	}
}

// 매칭 결과를 수신하는 함수
func receiveMatchResponse(t *testing.T, conn *websocket.Conn) MatchResponse {
	var matchResp MatchResponse
	err := conn.ReadJSON(&matchResp)
	if err != nil {
		t.Fatalf("Failed to receive match response: %v", err)
	}
	return matchResp
}

func TestMatchWebSocketAPI(t *testing.T) {
	participantCount := 2
	sessionIDs := make([]string, participantCount)
	userIDs := make([]string, participantCount)
	conns := make([]*websocket.Conn, participantCount)

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

	for i := 0; i < participantCount; i++ {
		sendMatchRequest(t, conns[i], userIDs[i])
	}

	// 매칭 결과를 수신 (양쪽에서 확인)
	time.Sleep(2 * time.Second) // 매칭 시간이 필요할 경우 대기

	matchResp1 := receiveMatchResponse(t, conns[0])
	matchResp2 := receiveMatchResponse(t, conns[1])

	// 매칭 응답이 올바르게 반환되었는지 확인
	if matchResp1.RoomID != matchResp2.RoomID {
		t.Errorf("RoomID mismatch between user1 and user2. user1: %s, user2: %s", matchResp1.RoomID, matchResp2.RoomID)
	}

	log.Printf("Test Passed: Match successful in room %s", matchResp1.RoomID)
}
