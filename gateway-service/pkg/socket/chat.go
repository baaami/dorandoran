package socket

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// WebSocket 연결 처리
func (app *Config) HandleChatSocket(w http.ResponseWriter, r *http.Request) {
	// 컨텍스트 생성 및 취소 함수 정의
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// TODO: 다중 인스턴스 환경에서의 세션 관리나 메시지 전달을 위해 Redis 같은 중앙 집중식 저장소를 활용하는 것을 고려
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// 클라이언트가 정상적으로 연결을 끊었을 경우 처리
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			log.Error().Str("err", err.Error()).Msgf("Unexpected WebSocket close error")
		} else {
			log.Info().Msg("WebSocket connection closed by client")
		}
		return
	}

	// URL에서 유저 ID 가져오기
	userID := r.Header.Get("X-User-ID")
	app.RegisterChatClient(conn, userID)
	defer func() {
		app.UnRegisterChatClient(userID)
		conn.Close()
	}()

	// WaitGroup을 사용하여 모든 고루틴이 종료될 때까지 대기
	var wg sync.WaitGroup
	wg.Add(2) // 두 개의 고루틴 (listenChatEvent, pingPump)

	// 게임에 필요한 초기 정보 전달

	// 메시지 처리 고루틴
	go func() {
		defer wg.Done()
		app.listenChatEvent(ctx, conn, userID)
	}()

	// Ping 메시지 전송 고루틴
	go func() {
		defer wg.Done()
		app.pingPump(ctx, conn)
	}()

	// 모든 고루틴이 종료될 때까지 대기
	wg.Wait()
}

// 메시지 읽기 처리
func (app *Config) listenChatEvent(ctx context.Context, conn *websocket.Conn, userID string) {
	for {
		select {
		case <-ctx.Done():
			return // 컨텍스트가 취소되면 고루틴 종료
		default:
			_, msg, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
					log.Error().Str("err", err.Error()).Msgf("Unexpected WebSocket close error")
				} else {
					log.Info().Msg("WebSocket connection closed by client")
				}
				return
			}

			var wsMsg WebSocketMessage
			if err := json.Unmarshal(msg, &wsMsg); err != nil {
				log.Printf("Failed to unmarshal message: %v", err)
				continue
			}

			switch wsMsg.Type {
			case MessageTypeChat:
				app.handleChatType(wsMsg.Status, userID, wsMsg.Payload)
			case MessageTypeRoom:
				app.handleRoomType(wsMsg.Status, userID, wsMsg.Payload)
			case MessageTypeGame:
				app.handleGameType(wsMsg.Status, userID, wsMsg.Payload)
			}
		}
	}
}

// chat type 메시지 처리
func (app *Config) handleChatType(status, userID string, payload json.RawMessage) {
	switch status {
	case MessageStatusChatBroadCast:
		app.handleBroadCastMessage(payload, userID)
	}
}

// room type 메시지 처리
func (app *Config) handleRoomType(status, userID string, payload json.RawMessage) {
	switch status {
	case MessageStatusRoomJoin:
		app.handleJoinMessage(payload, userID)
	case MessageStatusRoomLeave:
		app.handleLeaveMessage(payload, userID)
	}
}

// game type 메시지 처리
func (app *Config) handleGameType(status, userID string, payload json.RawMessage) {
	switch status {

	default:
	}
}

// Register
func (app *Config) RegisterChatClient(conn *websocket.Conn, userID string) {
	client := &Client{
		Conn: conn,
		Send: make(chan interface{}, 256),
	}

	// 쓰기 고루틴 시작
	go client.writePump()

	app.ChatClients.Store(userID, client)
	log.Printf("User %s register chat server", userID)
}

// UnRegister
func (app *Config) UnRegisterChatClient(userID string) {
	if clientInterface, ok := app.ChatClients.Load(userID); ok {
		client := clientInterface.(*Client)

		// Send 채널 닫기
		close(client.Send)

		// Channel에서 유저 제거
		app.ChatClients.Delete(userID)

		log.Printf("User %s unregistered chat server", userID)
	}
}

func (c *Client) writePump() {
	defer func() {
		c.Conn.Close()
		log.Printf("[INFO] writePump for user %v exited", c.Conn.RemoteAddr())
	}()

	for {
		message, ok := <-c.Send
		if !ok {
			// 채널이 닫힌 경우 연결을 닫습니다.
			c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		// 메시지를 전송합니다.
		if err := c.Conn.WriteJSON(message); err != nil {
			log.Printf("[ERROR] Failed to write message: %v", err)
			return
		}
	}
}
