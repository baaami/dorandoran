package socket

import (
	"sync"
	"time"

	"github.com/baaami/dorandoran/broker/pkg/redis"
	"github.com/gorilla/websocket"
	amqp "github.com/rabbitmq/amqp091-go"
)

type ChatMessage struct {
	RoomID   string `json:"room_id"`
	SenderID string `json:"sender_id"`
	Message  string `json:"message"`
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
	MessageTypeChat       = "chat"
	MessageTypeMatch      = "match"
	MessageTypeRegister   = "register"
	MessageTypeUnRegister = "unregister"
	MessageTypeBroadCast  = "broadcast"
	MessageTypeJoin       = "join"
	MessageTypeLeave      = "leave"
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
