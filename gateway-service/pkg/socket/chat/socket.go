package chat

import (
	"sync"

	"github.com/baaami/dorandoran/broker/event"
	"github.com/baaami/dorandoran/broker/pkg/data"
	"github.com/baaami/dorandoran/broker/pkg/redis"
	"github.com/gorilla/websocket"
)

const (
	MessageKindMessage     = "message"
	MessageKindJoin        = "join"
	MessageKindLeave       = "leave"
	MessageKindCheckRead   = "check_read"
	MessageKindChatLastest = "chat_latest"
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

// Room Type (Push)
const (
	PushMessageStatusRoomInfo = "info"
)

// Match Type (Push)
const (
	PushMessageStatusMatchSuccess = "success"
)

type Client struct {
	Conn *websocket.Conn
	Send chan interface{}
}

type Config struct {
	Rooms        sync.Map // key: roomID, value: *sync.Map (key: userID, value: *Client)
	ChatClients  sync.Map // key: userID, value: *Client
	ChatEmitter  *event.Emitter
	RedisClient  *redis.RedisClient
	EventChannel chan data.WebSocketMessage // RabbitMQ 이벤트를 수신할 채널
}
