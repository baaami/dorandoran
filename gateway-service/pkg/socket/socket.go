package socket

import (
	"sync"

	"github.com/baaami/dorandoran/broker/pkg/redis"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	MessageTypeChat       = "chat"
	MessageTypeMatch      = "match"
	MessageTypeRegister   = "register"
	MessageTypeUnRegister = "unregister"
	MessageTypeBroadCast  = "broadcast"
	MessageTypeJoin       = "join"
	MessageTypeLeave      = "leave"
)

type Config struct {
	Rooms        sync.Map //
	ChatClients  sync.Map
	MatchClients sync.Map
	Rabbit       *amqp.Connection
	RedisClient  *redis.RedisClient
}
