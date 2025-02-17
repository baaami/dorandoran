// services/chat/cmd/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"solo/pkg/db"
	"solo/pkg/mq"
	"solo/pkg/redis"
	"solo/services/chat/event"
	"solo/services/chat/handler"
	"solo/services/chat/repo"

	"solo/services/chat/service"
	"solo/services/chat/transport"
)

const webPort = 80

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	mongoClient, err := db.ConnectMongo()
	if err != nil {
		log.Panic("MongoDB 연결 실패: ", err)
	}
	defer mongoClient.Disconnect(ctx)

	mqClient, err := mq.ConnectToRabbitMQ()
	if err != nil {
		log.Panic("RabbitMQ 연결 실패: ", err)
	}
	defer mqClient.Conn.Close()

	redisClient, err := redis.NewRedisClient()
	if err != nil {
		log.Panic("Redis 연결 실패: ", err)
	}
	defer redisClient.Client.Close()

	emitter := event.NewEmitter(mqClient)

	chatRepo := repo.NewChatRepository(mongoClient)                       // Repository 생성
	chatService := service.NewChatService(chatRepo, redisClient, emitter) // Service 생성
	chatHandler := handler.NewChatHandler(chatService)                    // Handler 생성

	eventConsumer := event.NewConsumer(mqClient, redisClient, chatService)
	go eventConsumer.StartListening()

	router := transport.NewRouter(chatHandler)

	log.Printf("🚀 Chat Service Started on Port %d", webPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", webPort), router))
}
