package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"testing"
	"time"

	common "github.com/baaami/dorandoran/common/chat"
	data "github.com/baaami/dorandoran/common/user"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

const (
	API_GATEWAY_URL = "http://localhost:2719"
	WS_CHAT_URL     = "ws://localhost:2719/ws/chat"
	RoomID          = "1" // 임의의 RoomID
)

// 로그인 API를 호출하여 세션 ID와 유저 ID를 발급받는 함수
func loginAndGetSessionIDAndUserID(accessToken string) (string, string, error) {
	// 로그인 요청 데이터 설정 (필요한 데이터로 수정)
	loginData := map[string]string{
		"accessToken": accessToken,
	}
	reqBody, _ := json.Marshal(loginData)

	// 로그인 API 호출
	resp, err := http.Post(API_GATEWAY_URL+"/auth/kakao", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	// 세션 ID와 유저 ID 가져오기
	userID := ""
	sessionID := ""
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "session_id" {
			sessionID = cookie.Value
		}
	}

	var user data.User
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		return "", "", err
	}

	userID = strconv.Itoa(user.ID)

	return sessionID, userID, nil
}

// WebSocket을 통한 메시지 송수신 테스트
func TestChatBetweenTwoClients(t *testing.T) {
	// 1. 로그인 후 두 명의 클라이언트 세션 ID 및 유저 ID 발급
	sessionID1, userID1, err := loginAndGetSessionIDAndUserID("masterkey-2")
	assert.NoError(t, err)
	sessionID2, userID2, err := loginAndGetSessionIDAndUserID("masterkey-3")
	assert.NoError(t, err)

	// 2. 두 클라이언트가 WebSocket으로 접속
	conn1, err := connectWebSocket(t, sessionID1, userID1)
	assert.NoError(t, err)
	defer conn1.Close()

	conn2, err := connectWebSocket(t, sessionID2, userID2)
	assert.NoError(t, err)
	defer conn2.Close()

	// 3. 각 클라이언트에서 수신 메시지 대기하는 goroutine 실행
	go receiveChatMessages(t, conn1, userID1)
	go receiveChatMessages(t, conn2, userID2)

	// 4. 10번 대화를 주고받기
	for i := 0; i < 10; i++ {
		sendChatMessage(t, conn1, userID1, userID2, "Message from User1")
		time.Sleep(1 * time.Second) // 타임아웃을 두어 메시지 순차 송수신

		sendChatMessage(t, conn2, userID2, userID1, "Message from User2")
		time.Sleep(1 * time.Second)
	}
}

// WebSocket 서버에 접속하는 함수
func connectWebSocket(t *testing.T, sessionID string, userID string) (*websocket.Conn, error) {
	header := http.Header{}
	header.Add("Cookie", "session_id="+sessionID)
	header.Add("X-User-ID", userID)

	conn, _, err := websocket.DefaultDialer.Dial(WS_CHAT_URL, header)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	return conn, nil
}

// WebSocket으로 메시지를 보내는 함수
func sendChatMessage(t *testing.T, conn *websocket.Conn, senderID string, receiverID string, message string) {
	chatMessage := common.ChatMessage{
		RoomID:     RoomID,
		SenderID:   senderID,
		ReceiverID: receiverID,
		Message:    message,
	}

	log.Printf("sender, receiver: [%s, %s]", senderID, receiverID)

	wsMessage := common.WebSocketMessage{
		Type:    "chat",
		Payload: toJSONRawMessage(chatMessage),
	}

	err := conn.WriteJSON(wsMessage)
	assert.NoError(t, err, "Failed to send message")
}

// WebSocket에서 수신된 메시지를 처리하는 goroutine 함수
func receiveChatMessages(t *testing.T, conn *websocket.Conn, userID string) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("User %s disconnected: %v", userID, err)
			return
		}

		var chatMsg common.ChatMessage
		err = json.Unmarshal(msg, &chatMsg)
		assert.NoError(t, err)

		log.Printf("User %s received message from %s: %s", chatMsg.ReceiverID, chatMsg.SenderID, chatMsg.Message)
	}
}

// JSON을 RawMessage로 변환하는 유틸리티 함수
func toJSONRawMessage(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return json.RawMessage(data)
}
