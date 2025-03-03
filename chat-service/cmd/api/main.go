package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/baaami/dorandoran/chat/pkg/data"
	"github.com/baaami/dorandoran/chat/pkg/event"
	"github.com/baaami/dorandoran/chat/pkg/manager"
	"github.com/baaami/dorandoran/chat/pkg/redis"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	webPort  = "80"
	mongoURL = "mongodb://mongo:27017"
)

var client *mongo.Client

type Config struct {
	Models       data.Models
	Rabbit       *amqp.Connection
	Emitter      *event.Emitter
	RoomManager  *manager.RoomManager
	EventChannel chan event.EventPayload
}

func main() {
	// RabbitMQ 연결
	rabbitConn, err := connectToRabbitMQ()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer rabbitConn.Close()

	// MongoDB 연결
	mongoClient, err := connectToMongo()
	if err != nil {
		log.Panic(err)
	}
	client = mongoClient

	// MongoDB 연결 해제 시 사용되는 컨텍스트 생성
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// MongoDB 연결 해제
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	exchanges := []event.ExchangeConfig{
		{Name: event.ExchangeAppTopic, Type: "topic"},
		{Name: event.ExchangeChatRoomCreateEvents, Type: "fanout"},
		{Name: event.ExchangeCoupleRoomCreateEvents, Type: "fanout"},
	}

	emitter, err := event.NewEmitter(rabbitConn, exchanges)
	if err != nil {
		log.Printf("Failed to make new event emitter: %v", err)
		os.Exit(1)
	}

	// Redis 연결
	redisClient, err := redis.NewRedisClient()
	if err != nil {
		log.Printf("Failed to initialize Redis client: %v", err)
		os.Exit(1)
	}
	defer redisClient.Client.Close()

	models := data.New(client)

	// RoomManager 초기화
	roomManager := manager.NewRoomManager(redisClient, emitter, models)

	eventChannel := make(chan event.EventPayload, 10)

	// Config 구조체 초기화
	app := Config{
		Models:       models,
		Rabbit:       rabbitConn,
		Emitter:      emitter,
		RoomManager:  roomManager,
		EventChannel: eventChannel,
	}

	routingConfigs := []event.RoutingConfig{
		{
			Exchange: event.ExchangeConfig{Name: event.ExchangeAppTopic, Type: "topic"},
			Keys:     []string{event.EventTypeRoomTimeout, event.EventTypeFinalChoiceTimeout},
		},
		{
			Exchange: event.ExchangeConfig{Name: event.ExchangeMatchEvents, Type: "fanout"},
			Keys:     []string{}, // fanout 타입은 라우팅 키가 필요 없음
		},
	}

	consumer, err := event.NewConsumer(rabbitConn, routingConfigs)
	if err != nil {
		log.Printf("Failed to make new match consumer: %v", err)
		os.Exit(1)
	}

	// 핸들러 설정
	handlers := map[string]event.MessageHandler{
		event.EventTypeMatch:              event.MatchEventHandler,
		event.EventTypeRoomTimeout:        event.RoomTimeoutHandler,
		event.EventTypeFinalChoiceTimeout: event.FinalChoiceTimeoutHandler,
	}

	go func() {
		if err := consumer.Listen(handlers, eventChannel); err != nil {
			log.Printf("Failed to start RabbitMQ consumer: %v", err)
			os.Exit(1)
		}
	}()

	go app.RoomManager.MonitorRoomTimeouts()

	go app.EventPayloadHandler()

	// 웹 서버 시작
	log.Println("Starting Chat Service on port", webPort)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.routes(),
	}

	err = srv.ListenAndServe()
	if err != nil {
		log.Panic()
	}
}

// MongoDB에 연결하는 함수
func connectToMongo() (*mongo.Client, error) {
	// MongoDB 연결 옵션 설정
	clientOptions := options.Client().ApplyURI(mongoURL)
	clientOptions.SetAuth(options.Credential{
		Username: "admin",
		Password: "sample",
	})

	// MongoDB에 연결
	c, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Println("Error connecting to MongoDB:", err)
		return nil, err
	}

	log.Println("Connected to MongoDB!")

	return c, nil
}

func connectToRabbitMQ() (*amqp.Connection, error) {
	return amqp.Dial("amqp://guest:guest@rabbitmq")
}
