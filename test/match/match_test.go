package main

import (
	"log"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

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

// WebSocket 서버와 연결하는 함수
func connectToWebSocket(t *testing.T, userID string) *websocket.Conn {
	wsURL := "ws://localhost:2719/ws" // 서버 WebSocket 주소

	// WebSocket 연결 생성
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket server: %v", err)
	}
	log.Printf("User %s connected to WebSocket", userID)

	// Register 메시지 전송 (유저 등록)
	registerMsg := WebSocketMessage{
		Type: "register",
		Payload: map[string]string{
			"user_id": userID,
		},
	}
	err = conn.WriteJSON(registerMsg)
	if err != nil {
		t.Fatalf("Failed to send register message for user %s: %v", userID, err)
	}

	return conn
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
	// 유저1과 유저2 WebSocket 연결 생성
	conn1 := connectToWebSocket(t, "user1")
	defer func() {
		conn1.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		conn1.Close()
	}()

	conn2 := connectToWebSocket(t, "user2")
	defer func() {
		conn2.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		conn2.Close()
	}()

	// 유저1과 유저2 매칭 요청 전송
	sendMatchRequest(t, conn1, "user1")
	sendMatchRequest(t, conn2, "user2")

	// 매칭 결과를 수신 (양쪽에서 확인)
	time.Sleep(2 * time.Second) // 매칭 시간이 필요할 경우 대기

	matchResp1 := receiveMatchResponse(t, conn1)
	matchResp2 := receiveMatchResponse(t, conn2)

	// 매칭 응답이 올바르게 반환되었는지 확인
	if matchResp1.RoomID != matchResp2.RoomID {
		t.Errorf("RoomID mismatch between user1 and user2. user1: %s, user2: %s", matchResp1.RoomID, matchResp2.RoomID)
	}

	log.Printf("Test Passed: Match successful in room %s", matchResp1.RoomID)
}
