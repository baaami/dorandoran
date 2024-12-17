package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/baaami/dorandoran/match-socket-service/pkg/types"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type WebSocketMessage struct {
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload"`
}

type MatchResponse struct {
	Type   string `json:"type"`
	RoomID string `json:"room_id"`
}

const (
	MessageTypeMatch = "match"
)

const (
	PushMessageStatusMatchSuccess = "success"
	PushMessageStatusMatchFailure = "fail"
)

func (app *Config) HandleMatchSocket(c echo.Context) error {
	// 30초 타임아웃 설정
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for simplicity
		},
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "WebSocket upgrade failed")
	}
	defer conn.Close()

	log.Println("WebSocket connection established")

	userID := c.Request().Header.Get("X-User-ID")

	matchFilter, err := GetMatchFilter(userID)
	if err != nil {
		log.Printf("Failed to get matchfilter, user: %s", userID)
		return err
	}

	user, err := GetUserInfo(userID)
	if err != nil {
		log.Printf("Failed to get GetUserInfo, user: %s", userID)
		return err
	}

	waitingUser := types.WaitingUser{
		ID:              user.ID,
		Gender:          user.Gender,
		Birth:           user.Birth,
		Address:         types.Address(user.Address),
		AddressRangeUse: matchFilter.AddressRangeUse,
		AgeGroupUse:     matchFilter.AgeGroupUse,
	}

	// Check if user already exists in the Redis queue
	exists, err := app.RedisClient.IsUserInQueue(waitingUser)
	if err != nil {
		log.Printf("Error checking user %s in queue: %v", userID, err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check user in queue")
	}
	if exists {
		log.Printf("User %s is already in the matching queue", userID)
		return echo.NewHTTPError(http.StatusConflict, "User already in matching queue")
	}

	// 매칭 서버에 사용자 등록
	if err := app.RegisterMatchClient(conn, waitingUser); err != nil {
		log.Printf("Failed to register user %s to queue: %v", waitingUser.ID, err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to register user in matching queue")
	}

	defer func() {
		if err := app.UnRegisterMatchClient(waitingUser); err != nil {
			log.Printf("Failed to remove user %s from queue: %v", userID, err)
		}
		conn.Close()
	}()

	for {
		// 컨텍스트의 타임아웃을 웹소켓 연결에 적용
		deadline, ok := ctx.Deadline()
		if ok {
			conn.SetReadDeadline(deadline)
		}

		select {
		case <-ctx.Done():
			// 30초 타임아웃 시 매칭 실패 메시지 전송
			if ctx.Err() == context.DeadlineExceeded {
				log.Printf("Matching timed out for user %s", userID)
				app.sendMatchFailureMessage(conn)
			}

			return echo.NewHTTPError(http.StatusGatewayTimeout, "Failed to match time out")
		default:
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
					log.Printf("Unexpected WebSocket close error: %v", err)
				} else if ctx.Err() == context.DeadlineExceeded || isTimeoutError(err) {
					log.Printf("WebSocket read timeout for user %s", userID)
					app.sendMatchFailureMessage(conn)
					continue
				} else {
					log.Println("WebSocket connection closed by client")
				}
				return echo.NewHTTPError(http.StatusOK, "WebSocket connection closed by client")
			}
		}
	}
}

// 타임아웃 에러 확인 함수
func isTimeoutError(err error) bool {
	netErr, ok := err.(net.Error)
	return ok && netErr.Timeout()
}

func (app *Config) RegisterMatchClient(conn *websocket.Conn, waitingUser types.WaitingUser) error {
	_, ok := app.MatchClients.Load(waitingUser.ID)
	if ok {
		log.Printf("User %s already registered match server", waitingUser.ID)
		return fmt.Errorf("user %s already registered match server", waitingUser.ID)
	}

	// 1. user id에 매핑되는 websocket conn 추가
	app.MatchClients.Store(waitingUser.ID, conn)

	// 2. Redis 내 매칭 대기열에 추가
	if err := app.RedisClient.AddUserToQueue(waitingUser); err != nil {
		log.Printf("Failed to add user %s to queue: %v", waitingUser.ID, err)
		return fmt.Errorf("Failed to add user %s to queue: %v", waitingUser.ID, err)
	}

	log.Printf("User %s added to waiting queue", waitingUser.ID)

	return nil
}

func (app *Config) UnRegisterMatchClient(waitingUser types.WaitingUser) error {
	// 1. user id에 매핑되는 websocket conn 제거
	app.MatchClients.Delete(waitingUser.ID)

	// 2. Redis 내 매칭 대기열에서 제거
	if err := app.RedisClient.RemoveUserFromQueue(waitingUser); err != nil {
		log.Printf("Failed to remove user %s from queue: %v", waitingUser.ID, err)
	}

	log.Printf("User %s removed from waiting queue", waitingUser.ID)

	return nil
}

func (app *Config) MonitorQueue(coupleCnt int) {
	matchTotalNum := coupleCnt * 2 // 총 매칭 인원 수

	for {
		// Redis에서 대기열 모니터링 처리
		matchIDList, err := app.RedisClient.MonitorAndPopMatchingUsers(coupleCnt)
		if err != nil || len(matchIDList) < matchTotalNum {
			time.Sleep(2 * time.Second)
			continue
		}

		// 매칭 성공 시 사용자 알림 및 방 생성
		if len(matchIDList) == matchTotalNum {
			roomID := uuid.New().String()
			log.Printf("Matched room %s", roomID)

			err = app.createRoom(roomID, matchIDList)
			if err != nil {
				log.Printf("Failed to create room, room id: %s, err: %v", roomID, err.Error())
			}

			app.sendMatchSuccessMessage(matchIDList, roomID)
		}

		// 일정 주기만큼 실행
		time.Sleep(2 * time.Second)
	}
}

// 매칭 성공 메시지 전송 함수
func (app *Config) sendMatchSuccessMessage(matchList []string, roomID string) {
	matchMsg := MatchResponse{
		Type:   PushMessageStatusMatchSuccess,
		RoomID: roomID,
	}

	payload, err := json.Marshal(matchMsg)
	if err != nil {
		log.Printf("Failed to marshal match response: %v", err)
		return
	}

	webSocketMsg := WebSocketMessage{
		Kind:    MessageTypeMatch,
		Payload: json.RawMessage(payload),
	}

	log.Printf("Match Notify Start!!!")

	for _, userID := range matchList {
		log.Printf("Try to notify user, %s", userID)

		if conn, ok := app.MatchClients.Load(userID); ok {
			err := conn.(*websocket.Conn).WriteJSON(webSocketMsg)
			if err != nil {
				log.Printf("Failed to notify user %s: %v", userID, err)
			} else {
				log.Printf("Notified %s about match in room %s", userID, roomID)
			}
		} else {
			log.Printf("User %s not connected", userID)
		}

	}

	log.Printf("Match Notify End!!!")
}

// 매칭 실패 메시지 전송 함수
func (app *Config) sendMatchFailureMessage(conn *websocket.Conn) {
	matchMsg := MatchResponse{
		Type:   PushMessageStatusMatchFailure,
		RoomID: "",
	}

	payload, err := json.Marshal(matchMsg)
	if err != nil {
		log.Printf("Failed to marshal match response: %v", err)
		return
	}

	webSocketMsg := WebSocketMessage{
		Kind:    MessageTypeMatch,
		Payload: json.RawMessage(payload),
	}

	if err := conn.WriteJSON(webSocketMsg); err != nil {
		log.Printf("Failed to send match failure message: %v", err)
	}
}

func (app *Config) createRoom(roomID string, matchList []string) error {
	// Room creation logic would typically involve API calls or database updates.
	log.Printf("Creating room %s for users: %v", roomID, matchList)
	return nil // Simulate successful room creation
}

// [Bridge user] 유저 매칭 필터 조회
func GetMatchFilter(userID string) (*types.MatchFilter, error) {
	var matchFilter types.MatchFilter

	// Matching 필터 획득
	client := http.Client{}

	req, err := http.NewRequest(http.MethodGet, "http://user-service/match/filter", nil)
	if err != nil {
		return nil, err
	}

	// 사용자 ID를 요청의 헤더에 추가
	req.Header.Set("X-User-ID", userID)

	// 요청 실행
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	err = json.Unmarshal(body, &matchFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &matchFilter, nil
}

// [Bridge user] 유저 정보 조회
func GetUserInfo(userID string) (*types.User, error) {
	var user types.User

	// Matching 필터 획득
	client := http.Client{}

	req, err := http.NewRequest(http.MethodGet, "http://user-service/find", nil)
	if err != nil {
		return nil, err
	}

	// 사용자 ID를 요청의 헤더에 추가
	req.Header.Set("X-User-ID", userID)

	// 요청 실행
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	err = json.Unmarshal(body, &user)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &user, nil
}
