package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"solo/pkg/types/stype"
	"solo/services/game/service"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

// GameHandler는 WebSocket 연결 및 메시지를 처리하는 핸들러
type GameHandler struct {
	gameService *service.GameService
}

// NewGameHandler는 GameHandler 인스턴스를 생성
func NewGameHandler(gameService *service.GameService) *GameHandler {
	return &GameHandler{
		gameService: gameService,
	}
}

// WebSocket 업그레이더 설정
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *GameHandler) HandleGameSocket(c echo.Context) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// WebSocket으로 업그레이드
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("❌ WebSocket 업그레이드 실패: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "WebSocket upgrade failed")
	}
	defer conn.Close()

	// X-User-ID 헤더 확인 및 변환
	xUserID := c.Request().Header.Get("X-User-ID")
	userID, err := strconv.Atoi(xUserID)
	if err != nil {
		log.Printf("❌ 잘못된 X-User-ID: %s", xUserID)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid X-User-ID")
	}

	// 클라이언트 등록
	client := &service.Client{
		Conn: conn,
		Send: make(chan interface{}, 256),
		Ctx:  ctx,
	}

	err = h.gameService.RegisterUserToGame(userID, client)
	if err != nil {
		log.Printf("❌ 사용자 등록 실패: %v", err)
		return err
	}
	defer h.gameService.UnRegisterUserFromGame(userID) // 종료 시 클린업

	log.Printf("✅ WebSocket 연결: User %d", userID)

	// Ping-Pong 감지 채널
	pongChannel := make(chan bool, 10)

	// WaitGroup을 사용하여 모든 고루틴 종료 대기
	var wg sync.WaitGroup
	wg.Add(2)

	// Ping-Pong 메커니즘
	go func() {
		defer wg.Done()
		h.pingPongHandler(ctx, cancel, conn, userID, pongChannel)
	}()

	// 메시지 처리 고루틴
	go func() {
		defer wg.Done()
		h.listenForMessages(ctx, cancel, conn, userID, pongChannel)
	}()

	wg.Wait()

	return nil
}

// listenForMessages - 클라이언트의 메시지를 수신하고 처리
func (h *GameHandler) listenForMessages(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn, userID int, pongChannel chan bool) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("Context canceled, exiting listenForMessages for user %d", userID)
			return
		default:
			_, msg, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
					log.Printf("⚠️ Unexpected WebSocket close error")
				} else {
					log.Printf("📴 WebSocket connection closed by user %d", userID)
				}
				cancel()
				return
			}

			// JSON 메시지 파싱
			var wsMsg stype.WebSocketMessage
			if err := json.Unmarshal(msg, &wsMsg); err != nil {
				log.Printf("❌ 메시지 파싱 실패: %v", err)
				continue
			}

			// 메시지 종류에 따른 핸들러 호출
			switch wsMsg.Kind {
			case stype.MessageKindPong:
				select {
				case pongChannel <- true:
				default:
					log.Println("Pong received but no ping waiting")
				}
			case stype.MessageKindMessage:
				h.handleMessage(wsMsg.Payload, userID)
			case stype.MessageKindJoin:
				h.handleJoinMessage(wsMsg.Payload, userID)
			case stype.MessageKindLeave:
				h.handleLeaveMessage(wsMsg.Payload, userID)
			case stype.MessageKindRoomTimeout:
				h.handleRoomTimeout(wsMsg.Payload, userID)
			case stype.MessageKindFinalChoice:
				h.handleFinalChoice(wsMsg.Payload, userID)
			default:
				log.Printf("❌ 알 수 없는 메시지 타입: %s", wsMsg.Kind)
			}
		}
	}
}

func (h *GameHandler) handleMessage(payload json.RawMessage, userID int) {
	var chatMsg stype.ChatMessage
	if err := json.Unmarshal(payload, &chatMsg); err != nil {
		log.Printf("❌ Failed to unmarshal chat message: %v", err)
		return
	}

	// GameService를 통해 메시지 브로드캐스트
	err := h.gameService.BroadcastMessage(chatMsg.RoomID, userID, chatMsg.Message, chatMsg.HeadCnt)
	if err != nil {
		log.Printf("❌ BroadcastMessage 실패: %v", err)
		return
	}

	log.Printf("💬 Message from %d in room %s: %s", userID, chatMsg.RoomID, chatMsg.Message)
}

// handleJoinMessage - 게임 입장 처리
func (h *GameHandler) handleJoinMessage(payload json.RawMessage, userID int) {
	var joinMsg stype.JoinRoomMessage
	if err := json.Unmarshal(payload, &joinMsg); err != nil {
		log.Printf("❌ Failed to unmarshal join message: %v", err)
		return
	}

	// GameService를 통해 게임방에 참가 처리
	err := h.gameService.JoinGameRoom(joinMsg.RoomID, userID)
	if err != nil {
		log.Printf("❌ JoinGameRoom 실패: %v", err)
		return
	}

	log.Printf("🎮 User %d joined room %s", userID, joinMsg.RoomID)
}

// handleLeaveMessage - 게임 나가기 처리
func (h *GameHandler) handleLeaveMessage(payload json.RawMessage, userID int) {
	var leaveMsg stype.LeaveRoomMessage
	if err := json.Unmarshal(payload, &leaveMsg); err != nil {
		log.Printf("❌ Failed to unmarshal leave message: %v", err)
		return
	}

	// GameService를 통해 방 나가기 처리
	err := h.gameService.LeaveGameRoom(leaveMsg.RoomID, userID)
	if err != nil {
		log.Printf("❌ LeaveGameRoom 실패: %v", err)
		return
	}

	log.Printf("🚪 User %d left room %s", userID, leaveMsg.RoomID)
}

// handleRoomTimeout - 방 타임아웃 액션 처리
func (h *GameHandler) handleRoomTimeout(payload json.RawMessage, userID int) {
	var roomTimeoutMsg stype.RoomTimeoutMessage
	if err := json.Unmarshal(payload, &roomTimeoutMsg); err != nil {
		log.Printf("❌ Room Timeout 메시지 파싱 실패: %v", err)
		return
	}

	err := h.gameService.ProcessRoomTimeoutMessage(roomTimeoutMsg, userID)
	if err != nil {
		log.Printf("❌ Room Timeout 처리 실패: %v", err)
	}
}

func (h *GameHandler) handleFinalChoice(payload json.RawMessage, userID int) {
	var finalChoiceMsg stype.FinalChoiceMessage
	if err := json.Unmarshal(payload, &finalChoiceMsg); err != nil {
		log.Printf("❌ Final Choice 메시지 파싱 실패: %v", err)
		return
	}

	err := h.gameService.ProcessFinalChoice(userID, finalChoiceMsg)
	if err != nil {
		log.Printf("❌ Final Choice 처리 실패: %v", err)
	}
}

// pingPongHandler - 클라이언트 연결 상태 유지
func (h *GameHandler) pingPongHandler(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn, userID int, pongChannel chan bool) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingMessage := stype.WebSocketMessage{Kind: stype.MessageKindPing, Payload: nil}
			if err := conn.WriteJSON(pingMessage); err != nil {
				log.Printf("❌ Ping 전송 실패: User %d, Error: %v", userID, err)
				cancel()
				return
			}

			select {
			case <-pongChannel:
			case <-time.After(7 * time.Second):
				log.Printf("⏳ Pong 시간 초과: User %d", userID)
				cancel()
				return
			}
		}
	}
}
