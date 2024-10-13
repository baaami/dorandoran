package socket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/baaami/dorandoran/broker/pkg/redis"
	"github.com/gorilla/websocket"
	amqp "github.com/rabbitmq/amqp091-go"
)

type WebSocketMessage struct {
	Type    string          `json:"type"`
	Status  string          `json:"status"`
	Payload json.RawMessage `json:"payload"`
}

type Chat struct {
	RoomID    string    `bson:"room_id" json:"room_id"`
	SenderID  string    `bson:"sender_id" json:"sender_id"`
	Message   string    `bson:"message" json:"message"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

type ChatMessage struct {
	RoomID  string `json:"room_id"`
	Message string `json:"message"`
}

type ChatRoom struct {
	ID         string    `bson:"id" json:"id"` // UUID 사용
	Users      []string  `bson:"users" json:"users"`
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
	ModifiedAt time.Time `bson:"modified_at" json:"modified_at"`
	// 추가적으로 각 사용자의 마지막 확인 메시지 ID를 추적하기 위한 필드를 고려할 수 있음
	UserLastRead map[string]time.Time `bson:"user_last_read" json:"user_last_read"`
}

const (
	MessageTypeChat  = "chat"
	MessageTypeMatch = "match"
	MessageTypeRoom  = "room"
	MessageTypeGame  = "game"
)

// Chat Type (Receive)
const (
	MessageStatusChatBroadCast = "broadcast"
)

// Room Type (Receive)
const (
	MessageStatusRoomJoin  = "join"
	MessageStatusRoomLeave = "leave"
)

// Game Type (Receive)
const (
	MessageStatusGameFirstImpressionVote = "first_impression_vote" // 첫인상 투표
	MessageStatusGameSecretChatRequest   = "secret_chat_request"   // 비밀 채팅권 사용
	MessageStatusGameFinalSelection      = "final_selection"       // 최종 선택
)

const (
	MessageStatusMatchSuccess = "success"
)

// Game Type (Push)
const (
	PushMessageStatusGameStart                     = "start"
	PushMessageStatusGameIntroduceStart            = "introduce_start"
	PushMessageStatusGameIntroduceTurn             = "introduce_turn"
	PushMessageStatusGameIntroduceEnd              = "introduce_end"
	PushMessageStatusGameFirstImpressionVoteStart  = "first_impression_vote_start"
	PushMessageStatusGameFirstImpressionVoteEnd    = "first_impression_vote_end"
	PushMessageStatusGameFirstImpressionVoteResult = "first_impression_vote_result"
	PushMessageStatusGameMiniGameStart             = "mini_game_start"
	PushMessageStatusGameMiniGameEnd               = "mini_game_end"
	PushMessageStatusGameSecretChatRoomCreated     = "secret_chat_room_created"
	PushMessageStatusGameFinalSelectionStart       = "final_selection_start"
	PushMessageStatusGameFinalSelectionEnd         = "final_selection_end"
	PushMessageStatusGameFinalSelectionResult      = "final_selection_result"
	PushMessageStatusGameEnd                       = "end"
)

type Client struct {
	Conn *websocket.Conn
	Send chan interface{}
}

type Config struct {
	Rooms        sync.Map // key: roomID, value: *sync.Map (key: userID, value: *Client)
	ChatClients  sync.Map // key: userID, value: *Client
	MatchClients sync.Map
	Rabbit       *amqp.Connection
	RedisClient  *redis.RedisClient
}

const (
	pingPeriod = 60 * time.Second
	pongWait   = 70 * time.Second
	writeWait  = 10 * time.Second
)

// Ping 메시지 전송
func (app *Config) pingPump(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return // 컨텍스트가 취소되면 종료
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Failed to send ping message: %v", err)
				return
			}
		}
	}
}
