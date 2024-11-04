// gateway-service/cmd/api/main.go
package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/baaami/dorandoran/broker/event"
	"github.com/baaami/dorandoran/broker/pkg/redis"
	"github.com/baaami/dorandoran/broker/pkg/socket/chat"
	"github.com/baaami/dorandoran/broker/pkg/socket/match"

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

	// WebSocket 설정
	chatWSConfig := &chat.Config{
		Rooms:       sync.Map{},
		ChatClients: sync.Map{},
		ChatEmitter: &chatEmitter,
		RedisClient: redisClient,
	}

	matchWSConfig := &match.Config{
		MatchClients: sync.Map{},
		RedisClient:  redisClient,
	}

	// Redis 대기열 모니터링 고루틴 실행
	for copuleCnt := 1; copuleCnt <= coupleMaxCount; copuleCnt++ {
		maxRetry := 3
		go matchWSConfig.MonitorQueue(copuleCnt, maxRetry)
	}

	log.Info().Msgf("Starting Gateway service on port %d", webPort)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: app.routes(chatWSConfig, matchWSConfig),
	}

	err = srv.ListenAndServe()
	if err != nil {
		log.Error().Msgf("Error starting server: %v", err)
	}
}

func connect() (*amqp.Connection, error) {
	return amqp.Dial("amqp://guest:guest@rabbitmq")
}
