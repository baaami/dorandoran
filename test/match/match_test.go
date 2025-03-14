package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

const (
	API_GATEWAY_URL = "http://localhost:2719"
	WS_MATCH_URL    = "ws://localhost:2720/ws/match"
)

type Address struct {
	City     string `gorm:"size:100" json:"city"`
	District string `gorm:"size:100" json:"district"`
	Street   string `gorm:"size:100" json:"street"`
}

type User struct {
	ID         int     `gorm:"primaryKey;autoIncrement" json:"id"`
	SnsType    int     `gorm:"index" json:"sns_type"`
	SnsID      string  `gorm:"index" json:"sns_id"`
	Name       string  `gorm:"size:100" json:"name"`
	Gender     int     `json:"gender"`
	Birth      string  `gorm:"size:20" json:"birth"`
	Address    Address `gorm:"embedded;embeddedPrefix:address_" json:"address"`
	GameStatus int     `gorm:"default:0" json:"game_status"`
	GameRoomID string  `gorm:"size:100" json:"game_room_id"`
	GamePoint  int     `json:"game_point"`
}

// WebSocketMessage 구조체 정의
type WebSocketMessage struct {
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload"`
}

// MatchResponse 구조체 정의
type MatchResponse struct {
	Type   string `json:"type"`
	RoomID string `json:"room_id"`
}

type UnRegisterMessage struct {
	UserID string `json:"user_id"`
}

// 로그인 API를 호출하여 세션 ID와 유저 ID를 발급받는 함수
func loginAndGetSessionIDAndUserID(accessToken string) (string, string, error) {
	loginData := map[string]string{
		"accessToken": accessToken,
	}
	reqBody, err := json.Marshal(loginData)
	if err != nil {
		log.Printf("[ERROR] Error marshaling login data: %v", err)
		return "", "", err
	}

	resp, err := http.Post(API_GATEWAY_URL+"/auth/kakao", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Printf("[ERROR] Error sending login request: %v", err)
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[ERROR] Login request failed, status code: %d", resp.StatusCode)
		return "", "", fmt.Errorf("login failed with status code: %d", resp.StatusCode)
	}

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
	userID := strconv.Itoa(user.ID)

	return sessionID, userID, nil
}

func updateProfile(t *testing.T, sessionID string, userID string, gender int, birth string, city string) error {
	client := http.Client{}

	updatedUser := User{
		Gender: gender,
		Birth:  birth,
		Address: Address{
			City: city,
		},
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
		return fmt.Errorf("failed to send request to doran-user: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create user, status code: %d", resp.StatusCode)
	}

	return nil
}

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

func receiveMatchResponse(t *testing.T, conn *websocket.Conn) WebSocketMessage {
	var webSocketMsg WebSocketMessage
	err := conn.ReadJSON(&webSocketMsg)
	if err != nil {
		t.Fatalf("Failed to receive match response: %v", err)
	}

	var matchResp MatchResponse
	err = json.Unmarshal(webSocketMsg.Payload, &matchResp)
	if err != nil {
		t.Fatalf("Failed to unmarshal MatchResponse: %v", err)
	}

	if matchResp.Type == "fail" {
		t.Logf("Received match failure message: %v", webSocketMsg)
	}

	return webSocketMsg
}

func TestMatchWebSocketAPI(t *testing.T) {
	participantCount := 8
	sessionIDs := make([]string, participantCount)
	userIDs := make([]string, participantCount)
	conns := make([]*websocket.Conn, participantCount)
	responseChan := make(chan WebSocketMessage, participantCount)

	cities := []string{"Seoul", "Busan", "Incheon", "Daegu"}
	rand.Seed(time.Now().UnixNano())

	// 1. 100명의 참가자가 로그인하여 세션 ID와 유저 ID 발급
	for i := 0; i < participantCount; i++ {
		accessToken := fmt.Sprintf("masterkey-%d", i+1)
		sessionID, userID, err := loginAndGetSessionIDAndUserID(accessToken)
		assert.NoError(t, err)
		sessionIDs[i] = sessionID
		userIDs[i] = userID
	}

	// 2. 100명의 참가자가 사용자 프로필 정보 입력
	for i := 0; i < participantCount; i++ {
		gender := i % 2 // 0 (남자), 1 (여자)
		birthYear := rand.Intn(2000-1980+1) + 1980
		birth := fmt.Sprintf("%d%02d%02d", birthYear, rand.Intn(12)+1, rand.Intn(28)+1)
		city := cities[rand.Intn(len(cities))]

		err := updateProfile(t, sessionIDs[i], userIDs[i], gender, birth, city)
		if err != nil {
			fmt.Printf("Fail to updateProfile, user: %s, err: %s\n", userIDs[i], err)
			continue
		}
	}

	// 3. 100명의 참가자가 WebSocket으로 접속
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

	// 매칭된 사람과 매칭되지 않은 사람들의 정보를 저장할 구조체 정의
	type UserInfo struct {
		UserID string
		Gender int
		Birth  string
	}

	// 매칭된 사람들과 매칭되지 않은 사람들을 저장할 리스트
	matchedUsers := make(map[string][]UserInfo) // roomID별로 매칭된 사람들 그룹화
	unmatchedUsers := []UserInfo{}

	// 5. 응답을 수신하고, 성공 또는 실패 메시지 확인
	matchResponses := make([]WebSocketMessage, participantCount)
	for i := 0; i < participantCount; i++ {
		matchResponses[i] = <-responseChan

		gender := i % 2
		birthYear := rand.Intn(2010-1980+1) + 1980                                      // 랜덤 출생 연도 (기존 프로필 정보에서 가져와야 함)
		birth := fmt.Sprintf("%d%02d%02d", birthYear, rand.Intn(12)+1, rand.Intn(28)+1) // 예시 출생일

		userInfo := UserInfo{
			UserID: userIDs[i],
			Gender: gender,
			Birth:  birth,
		}

		var matchResp MatchResponse
		err := json.Unmarshal(matchResponses[i].Payload, &matchResp)
		assert.NoError(t, err)

		if matchResp.Type == "success" {
			if matchResp.RoomID != "" {
				t.Logf("%s User allocated room %s", userIDs[i], matchResp.RoomID)
				matchedUsers[matchResp.RoomID] = append(matchedUsers[matchResp.RoomID], userInfo)
			}
		} else if matchResp.Type == "fail" {
			t.Logf("%s User match failed due to timeout", userIDs[i])
			unmatchedUsers = append(unmatchedUsers, userInfo)
		}
	}

	// 매칭 결과 보고서 출력
	fmt.Println("\n--- Matching Report ---")
	fmt.Println("Matched Users by Room:")
	for roomID, users := range matchedUsers {
		fmt.Printf("\nRoom ID: %s\n", roomID)
		for _, user := range users {
			fmt.Printf("User ID: %s, Gender: %d, Birth: %s\n", user.UserID, user.Gender, user.Birth)
		}
	}

	fmt.Println("\nUnmatched Users:")
	for _, user := range unmatchedUsers {
		fmt.Printf("User ID: %s, Gender: %d, Birth: %s\n", user.UserID, user.Gender, user.Birth)
	}

	// 6. WebSocket 연결 닫기
	for _, conn := range conns {
		conn.Close()
	}
}
