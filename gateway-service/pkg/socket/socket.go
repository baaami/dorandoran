package socket

import (
	"sync"

	"github.com/baaami/dorandoran/broker/pkg/redis"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Config struct {
	ChatClients  sync.Map
	MatchClients sync.Map
	Rabbit       *amqp.Connection
	RedisClient  *redis.RedisClient
}
