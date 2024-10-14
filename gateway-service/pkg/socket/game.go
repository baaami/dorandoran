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
func (app *Config) HandleGameSocket(w http.ResponseWriter, r *http.Request) {
	// 컨텍스트 생성 및 취소 함수 정의
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

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
	app.registerGameClient(conn, userID)
	defer func() {
		app.unRegisterGameClient(userID)
		conn.Close()
	}()

	// WaitGroup을 사용하여 모든 고루틴이 종료될 때까지 대기
	var wg sync.WaitGroup
	wg.Add(2)

	// 게임에 필요한 초기 정보 전달

	// 메시지 처리 고루틴
	go func() {
		defer wg.Done()
		app.listenGameEvent(ctx, conn, userID)
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
func (app *Config) listenGameEvent(ctx context.Context, conn *websocket.Conn, userID string) {
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
			case MessageTypeGame:
				app.handleGameType(wsMsg.Status, userID, wsMsg.Payload)
			}
		}
	}
}

// game type 메시지 처리
func (app *Config) handleGameType(status, userID string, payload json.RawMessage) {
	switch status {
	default:
	}
}

// Register
func (app *Config) registerGameClient(conn *websocket.Conn, userID string) {
	app.GameClients.Store(userID, conn)
	log.Printf("User %s register game server", userID)
}

// UnRegister
func (app *Config) unRegisterGameClient(userID string) {
	app.GameClients.Delete(userID)
	log.Printf("User %s unregister game server", userID)
}
