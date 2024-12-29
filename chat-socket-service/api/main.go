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

	exchanges := []event.ExchangeConfig{
		{Name: "app_topic", Type: "topic"},
		// TODO: match_events를 topic으로 하여 라우팅 키로 game, couple을 나눠도 될듯
		{Name: "match_events", Type: "fanout"},
	}

	// RabbitMQ Emitter 생성
	chatEmitter, err := event.NewEmitter(rabbitConn, exchanges)
	if err != nil {
		log.Fatalf("Failed to create RabbitMQ emitter: %v", err)
		os.Exit(1)
	}

	// 채널 생성: 이벤트 전달용
	chatEventChannel := make(chan types.WebSocketMessage, 100)

	app := &Config{
		RedisClient:  redisClient,
		ChatEmitter:  chatEmitter,
		EventChannel: chatEventChannel,
	}

	// RoutingConfig 설정
	routingConfigs := []event.RoutingConfig{
		{
			Exchange: event.ExchangeConfig{Name: event.ExchangeAppTopic, Type: "topic"},
			Keys:     []string{"chat", "chat.latest", "room.leave"},
		},
		{
			Exchange: event.ExchangeConfig{Name: event.ExchangeCoupleRoomCreateEvents, Type: "fanout"},
			Keys:     []string{}, // fanout 타입은 라우팅 키가 필요 없음
		},
	}

	// Consumer 생성
	chatConsumer, err := event.NewConsumer(rabbitConn, routingConfigs)
	if err != nil {
		log.Fatalf("Failed to create RabbitMQ consumer: %v", err)
	}

	// key: event type, value: handler function
	handlers := map[string]event.MessageHandler{
		event.EventTypeChat:             event.ChatMessageHandler,      // 채팅 메시지 핸들러
		event.EventTypeChatLatest:       event.ChatLatestHandler,       // 최신 채팅 핸들러
		event.EventTypeCoupleRoomCreate: event.CreateCoupleRoomHandler, // 커플 방 생성 핸들러
		event.EventTypeRoomLeave:        event.RoomLeaveHandler,
	}

	// RabbitMQ Consumer Listen 고루틴 실행
	go func() {
		log.Println("Starting RabbitMQ consumer for events")
		if err := chatConsumer.Listen(handlers, chatEventChannel); err != nil {
			log.Fatalf("Failed to start RabbitMQ consumer: %v", err)
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
