// gateway-service/cmd/api/main.go
package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/baaami/dorandoran/broker/event"
	"github.com/baaami/dorandoran/broker/pkg/data"
	"github.com/baaami/dorandoran/broker/pkg/redis"
	"github.com/baaami/dorandoran/broker/pkg/socket/chat"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

const webPort = 80
const coupleMaxCount = 4

type Config struct{}

func main() {
	// RabbitMQ 연결
	rabbitConn, err := connect()
	if err != nil {
		log.Error().Msg(err.Error())
		os.Exit(1)
	}
	defer rabbitConn.Close()

	// Redis 연결
	redisClient, err := redis.NewRedisClient()
	if err != nil {
		log.Error().Msgf("Failed to initialize Redis client: %v", err)
	}

	app := Config{}

	chatEmitter, err := event.NewEventEmitter(rabbitConn)
	if err != nil {
		log.Error().Msgf("Failed to make new event emitter: %v", err)
		os.Exit(1)
	}

	// 채널 생성: 이벤트 전달용
	chatEventChannel := make(chan data.WebSocketMessage, 100)

	// WebSocket 설정
	chatWSConfig := &chat.Config{
		Rooms:        sync.Map{},
		ChatClients:  sync.Map{},
		ChatEmitter:  &chatEmitter,
		RedisClient:  redisClient,
		EventChannel: chatEventChannel,
	}

	// RabbitMQ Consumer
	chatConsumer, err := event.NewConsumer(rabbitConn)
	if err != nil {
		log.Error().Msgf("Failed to make new event consumer: %v", err)
		os.Exit(1)
	}

	// RabbitMQ Consumer Listen 고루틴 실행
	go func() {
		log.Info().Msg("Starting RabbitMQ consumer for chat.latest events")
		if err := chatConsumer.Listen([]string{"chat.latest", "room.remain.time", "room.timeout"}, chatEventChannel); err != nil {
			log.Error().Msgf("Failed to start RabbitMQ consumer: %v", err)
			os.Exit(1)
		}
	}()

	go chatWSConfig.SendSocketByChatEvents()

	log.Info().Msgf("Starting Gateway service on port %d", webPort)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: app.routes(redisClient),
	}

	err = srv.ListenAndServe()
	if err != nil {
		log.Error().Msgf("Error starting server: %v", err)
	}
}

func connect() (*amqp.Connection, error) {
	return amqp.Dial("amqp://guest:guest@rabbitmq")
}
