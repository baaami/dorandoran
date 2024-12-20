package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/baaami/dorandoran/match-socket-service/pkg/event"
	"github.com/baaami/dorandoran/match-socket-service/pkg/redis"
	"github.com/baaami/dorandoran/match-socket-service/pkg/types"

	amqp "github.com/rabbitmq/amqp091-go"
)

const webPort = 80

type Config struct {
	MatchClients sync.Map // key: userID (int), value: *websocket.Conn
	RedisClient  *redis.RedisClient
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

	// 채팅방 데이터 처리를 위한 채널 생성
	chatRoomChan := make(chan types.ChatRoom)

	chatRoomConsumer, err := event.NewConsumer(rabbitConn)
	if err != nil {
		log.Printf("Failed to make new match consumer: %v", err)
		os.Exit(1)
	}

	go func() {
		log.Println("Starting RabbitMQ consumer for chat_room_create_events exchange")
		if err := chatRoomConsumer.Listen(chatRoomChan); err != nil {
			log.Printf("Failed to start RabbitMQ consumer: %v", err)
			os.Exit(1)
		}
	}()

	// Config 구조체 생성
	app := Config{
		RedisClient: redisClient,
	}

	// 채팅방 데이터 처리
	go func() {
		for chatRoom := range chatRoomChan {
			// 사용자들에게 매칭 성공 메시지 전송
			app.sendMatchSuccessMessage(chatRoom.Users, chatRoom.ID)
		}
	}()

	// HTTP 서버 설정
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: app.routes(),
	}

	// 서버 시작
	log.Printf("Starting Match Socket Service on port %d", webPort)
	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func connect() (*amqp.Connection, error) {
	return amqp.Dial("amqp://guest:guest@rabbitmq")
}
