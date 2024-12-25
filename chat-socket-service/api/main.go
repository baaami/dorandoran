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
	// TODO: Rooms로 관리하는 부분은 redis를 통해서 관리하도록 수정해야함 -> stateless 서버를 위함
	Rooms        sync.Map       // key: roomID, value: *sync.Map (key: userID, value: *Client)
	ChatClients  sync.Map       // key: userID, value: *Client
	ChatEmitter  *event.Emitter // 채팅 데이터 발행을 위한 emitter
	RedisClient  *redis.RedisClient
	EventChannel chan types.WebSocketMessage // RabbitMQ 이벤트를 수신할 채널
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
		log.Fatalf("Failed to connect rabbitMQ client: %v", err)
		os.Exit(1)
	}
	defer rabbitConn.Close()

	chatEmitter, err := event.NewEventEmitter(rabbitConn)
	if err != nil {
		log.Printf("Failed to make new event emitter: %v", err)
		os.Exit(1)
	}

	// 채널 생성: 이벤트 전달용
	chatEventChannel := make(chan types.WebSocketMessage, 100)

	app := &Config{
		Rooms:        sync.Map{},
		ChatClients:  sync.Map{},
		ChatEmitter:  &chatEmitter,
		RedisClient:  redisClient,
		EventChannel: chatEventChannel,
	}

	// // RabbitMQ Consumer
	chatConsumer, err := event.NewConsumer(rabbitConn)
	if err != nil {
		log.Printf("Failed to make new event consumer: %v", err)
		os.Exit(1)
	}

	// // RabbitMQ Consumer Listen 고루틴 실행
	go func() {
		log.Println("Starting RabbitMQ consumer for chat.latest events")
		if err := chatConsumer.Listen([]string{"chat.latest"}, chatEventChannel); err != nil {
			log.Printf("Failed to start RabbitMQ consumer: %v", err)
			os.Exit(1)
		}
	}()

	go app.SendSocketByChatEvents()

	log.Printf("Starting Gateway service on port %d", webPort)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: app.routes(),
	}

	err = srv.ListenAndServe()
	if err != nil {
		log.Printf("Error starting server: %v", err)
	}
}

func connect() (*amqp.Connection, error) {
	return amqp.Dial("amqp://guest:guest@rabbitmq")
}
