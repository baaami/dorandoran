package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/baaami/dorandoran/chat-socket-service/pkg/event"
	"github.com/baaami/dorandoran/chat-socket-service/pkg/redis"
	"github.com/baaami/dorandoran/chat-socket-service/pkg/types"

	amqp "github.com/rabbitmq/amqp091-go"
)

const webPort = 80

type Config struct {
	ChatClients  sync.Map                    // key: userID, value: *Client
	ChatEmitter  *event.Emitter              // RabbitMQ Producer
	RedisClient  *redis.RedisClient          // Redis 정보 관리
	EventChannel chan types.WebSocketMessage // RabbitMQ 메시지 소비용 채널
}

func main() {
	// Redis 연결
	redisClient, err := redis.NewRedisClient()
	if err != nil {
		log.Fatalf("Failed to initialize Redis client: %v", err)
		os.Exit(1)
	}

	// RabbitMQ 연결
	rabbitConn, err := connect()
	if err != nil {
		log.Fatalf("Failed to connect RabbitMQ: %v", err)
		os.Exit(1)
	}
	defer rabbitConn.Close()

	// RabbitMQ Emitter 생성
	chatEmitter, err := event.NewEventEmitter(rabbitConn)
	if err != nil {
		log.Fatalf("Failed to create RabbitMQ emitter: %v", err)
		os.Exit(1)
	}

	// 채널 생성: 이벤트 전달용
	chatEventChannel := make(chan types.WebSocketMessage, 100)

	app := &Config{
		RedisClient:  redisClient,
		ChatEmitter:  &chatEmitter,
		EventChannel: chatEventChannel,
	}

	// RabbitMQ Consumer
	chatConsumer, err := event.NewConsumer(rabbitConn)
	if err != nil {
		log.Printf("Failed to make new event consumer: %v", err)
		os.Exit(1)
	}

	// RabbitMQ Consumer Listen 고루틴 실행
	go func() {
		log.Println("Starting RabbitMQ consumer for chat.latest events")
		if err := chatConsumer.Listen([]string{"chat", "chat.latest"}, chatEventChannel); err != nil {
			log.Printf("Failed to start RabbitMQ consumer: %v", err)
			os.Exit(1)
		}
	}()

	go app.SendSocketByChatEvents()

	// HTTP 서버 시작
	log.Printf("Starting server on port %d", webPort)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: app.routes(),
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func connect() (*amqp.Connection, error) {
	return amqp.Dial("amqp://guest:guest@rabbitmq")
}
