package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"solo/pkg/types/commontype"
	"solo/services/match/service"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type MatchHandler struct {
	matchService *service.MatchService
}

func NewMatchHandler(matchService *service.MatchService) *MatchHandler {
	return &MatchHandler{matchService: matchService}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *MatchHandler) HandleMatchSocket(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "WebSocket upgrade failed")
	}
	defer conn.Close()

	xUserID := c.Request().Header.Get("X-User-ID")
	userID, err := strconv.Atoi(xUserID)
	if err != nil {
		log.Printf("User ID is not a number: %s", xUserID)
		return echo.NewHTTPError(http.StatusInternalServerError, "User ID is not a number")
	}

	user, err := GetUserInfo(userID)
	if err != nil {
		log.Printf("Failed to get GetUserInfo, user: %d", userID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get GetUserInfo")
	}

	userFilter, err := GetMatchFilterInfo(userID)
	if err != nil {
		log.Printf("Failed to get GetMatchFilterInfo, user: %d", userID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get GetMatchFilterInfo")
	}

	waitingUser := commontype.WaitingUser{
		ID:          user.ID,
		Gender:      user.Gender,
		Birth:       user.Birth,
		Address:     commontype.Address(user.Address),
		CoupleCount: userFilter.CoupleCount,
	}

	err = h.matchService.RegisterUserToMatch(conn, waitingUser)
	if err != nil {
		log.Printf("Failed to register user %d to queue: %v", userID, err)
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to register user to queue")
	}
	defer h.matchService.UnregisterUserFromMatch(waitingUser)

	for {
		deadline, ok := ctx.Deadline()
		if ok {
			conn.SetReadDeadline(deadline)
		}

		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				log.Printf("Matching timed out for user %d", userID)
				h.matchService.SendMatchFailureMessage(conn)
			}
			return nil
		default:
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
					log.Printf("Unexpected WebSocket close error: %v", err)
				} else if ctx.Err() == context.DeadlineExceeded || isTimeoutError(err) {
					log.Printf("WebSocket read timeout for user %d", userID)
					h.matchService.SendMatchFailureMessage(conn)
					continue
				} else {
					log.Printf("WebSocket connection closed by client, user id: %d", userID)
				}
				return nil
			}
		}
	}
}

// 타임아웃 에러 확인 함수
func isTimeoutError(err error) bool {
	netErr, ok := err.(net.Error)
	return ok && netErr.Timeout()
}

// [Bridge user] 유저 정보 조회
func GetUserInfo(userID int) (*commontype.User, error) {
	var user commontype.User

	// Matching 필터 획득
	client := http.Client{}

	req, err := http.NewRequest(http.MethodGet, "http://user-service/find", nil)
	if err != nil {
		return nil, err
	}

	// 사용자 ID를 요청의 헤더에 추가
	req.Header.Set("X-User-ID", strconv.Itoa(userID))

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

// [Bridge user] 유저 필터 정보 조회
func GetMatchFilterInfo(userID int) (*commontype.MatchFilter, error) {
	var matchFilter commontype.MatchFilter

	// Matching 필터 획득
	client := http.Client{}

	req, err := http.NewRequest(http.MethodGet, "http://user-service/match/filter", nil)
	if err != nil {
		return nil, err
	}

	// 사용자 ID를 요청의 헤더에 추가
	req.Header.Set("X-User-ID", strconv.Itoa(userID))

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
