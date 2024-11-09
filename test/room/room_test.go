// chat_test.go

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

const (
	API_GATEWAY_URL = "http://localhost:2719"
	WS_CHAT_URL     = "ws://localhost:2719/ws/chat"
	RoomID          = "605c460d-2b43-418b-bbb3-8bff1955e1a8" // 테스트용 RoomID
)

type WebSocketMessage struct {
	Type    string          `json:"type"`
	Status  string          `json:"status"`
	Payload json.RawMessage `json:"payload"`
}

type Chat struct {
	Type      string    `bson:"type" json:"type"`
	RoomID    string    `bson:"room_id" json:"room_id"`
	SenderID  int       `bson:"sender_id" json:"sender_id"`
	Message   string    `bson:"message" json:"message"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

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

	conn, _, err := websocket.DefaultDialer.Dial(WS_CHAT_URL, header)
	if err != nil {
		log.Printf("[ERROR] Failed to connect to WebSocket for User %s: %v", userID, err)
		return nil, err
	}
	log.Printf("[INFO] User %s connected to WebSocket", userID)
	return conn, nil
}

// WebSocket으로 메시지를 보내는 함수
func sendChat(t *testing.T, conn *websocket.Conn, senderID string, message string) {
	nSenderID, _ := strconv.Atoi(senderID)

	Chat := Chat{
		RoomID:   RoomID,
		SenderID: nSenderID,
		Message:  message,
	}

	wsMessage := WebSocketMessage{
		Type:    "chat",
		Status:  "broadcast",
		Payload: toJSONRawMessage(Chat),
	}

	err := conn.WriteJSON(wsMessage)
	if err != nil {
		log.Printf("[ERROR] Failed to send message from User %s: %v", senderID, err)
	}
}

// WebSocket에서 수신된 메시지를 처리하는 goroutine 함수
func receiveChats(t *testing.T, conn *websocket.Conn, userID string, wg *sync.WaitGroup) {
	defer wg.Done()
	for {

		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[INFO] User %s disconnected: %v", userID, err)
			return
		}

		var webSocketMsg WebSocketMessage
		err = json.Unmarshal(msg, &webSocketMsg)
		if err != nil {
			log.Printf("[ERROR] Failed to unmarshal websocket message for User %s: %v", userID, err)
			continue
		}

		var chatMsg Chat
		err = json.Unmarshal(webSocketMsg.Payload, &chatMsg)
		if err != nil {
			log.Printf("[ERROR] Failed to unmarshal chat message for User %s: %v", userID, err)
			continue
		}

		log.Printf("[INFO] User %s received message from %d: %s", userID, chatMsg.SenderID, chatMsg.Message)
	}
}

// JSON을 RawMessage로 변환하는 유틸리티 함수
func toJSONRawMessage(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return json.RawMessage(data)
}

// 5명의 참가자가 채팅을 테스트하는 함수
func TestChatAmongFiveClients(t *testing.T) {
	participantCount := 8
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

	// 3. 각 참가자들이 100번 방에 참여
	var joinWg sync.WaitGroup
	joinWg.Add(participantCount)
	for i := 0; i < participantCount; i++ {
		go func(i int) {
			defer joinWg.Done()
			joinRoom(t, conns[i], userIDs[i])
		}(i)
	}
	joinWg.Wait()
	log.Println("[INFO] All participants have joined the room")

	// 4. 각 참가자들이 수신 메시지 대기하는 goroutine 실행
	var receiveWg sync.WaitGroup
	receiveWg.Add(participantCount)
	for i := 0; i < participantCount; i++ {
		go receiveChats(t, conns[i], userIDs[i], &receiveWg)
	}

	// 수신 메시지 대기
	time.Sleep(2 * time.Second)

	// 5. 각 참가자들이 채팅 메시지 1개씩 송신
	for i := 0; i < participantCount; i++ {
		message := fmt.Sprintf("Hello from User %s", userIDs[i])
		sendChat(t, conns[i], userIDs[i], message)
	}

	// 메시지가 모두 전달될 시간을 기다림
	time.Sleep(3 * time.Second)

	// 6. 모든 참가자들이 방을 떠남
	var leaveWg sync.WaitGroup
	leaveWg.Add(participantCount)
	for i := 0; i < participantCount; i++ {
		go func(i int) {
			defer leaveWg.Done()
			leaveRoom(t, conns[i], userIDs[i])
		}(i)
	}
	leaveWg.Wait()
	log.Println("[INFO] All participants have left the room")

	// 7. 참가자들의 WebSocket 연결 종료
	for i := 0; i < participantCount; i++ {
		conns[i].Close()
		log.Printf("[INFO] User %s WebSocket connection closed", userIDs[i])
	}

	// 수신 메시지 처리 goroutine들이 종료될 때까지 기다림
	receiveWg.Wait()
	log.Println("[INFO] Test completed")
}

// 참가자가 방에 참여하는 함수
func joinRoom(t *testing.T, conn *websocket.Conn, userID string) {
	joinMsg := WebSocketMessage{
		Type:   "room",
		Status: "join",
		Payload: toJSONRawMessage(map[string]string{
			"room_id": RoomID,
		}),
	}

	err := conn.WriteJSON(joinMsg)
	if err != nil {
		log.Printf("[ERROR] User %s failed to join room: %v", userID, err)
	} else {
		log.Printf("[INFO] User %s joined room %s", userID, RoomID)
	}
}

// 참가자가 방을 떠나는 함수
func leaveRoom(t *testing.T, conn *websocket.Conn, userID string) {
	leaveMsg := WebSocketMessage{
		Type:   "room",
		Status: "leave",
		Payload: toJSONRawMessage(map[string]string{
			"room_id": RoomID,
		}),
	}

	err := conn.WriteJSON(leaveMsg)
	if err != nil {
		log.Printf("[ERROR] User %s failed to leave room: %v", userID, err)
	} else {
		log.Printf("[INFO] User %s left room %s", userID, RoomID)
	}
}
