package socket

import (
	"sync"

	"github.com/baaami/dorandoran/broker/pkg/redis"
	"github.com/gorilla/websocket"
	amqp "github.com/rabbitmq/amqp091-go"
)

type ChatMessage struct {
	RoomID   string `json:"room_id"`
	SenderID string `json:"sender_id"`
	Message  string `json:"message"`
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
