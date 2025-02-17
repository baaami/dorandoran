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
	"solo/services/chat/repo"
	"solo/services/game/event"
	"solo/services/game/handler"
	"solo/services/game/service"
	"solo/services/game/transport"
)

const webPort = 80

func main() {
	// MongoDB 연결 해제 시 사용되는 컨텍스트 생성
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// MongoDB 연결
	mongoClient, err := db.ConnectMongo()
	if err != nil {
		log.Panic("MongoDB 연결 실패: ", err)
	}
	defer mongoClient.Disconnect(ctx)

	// RabbitMQ 연결
	mqClient, err := mq.ConnectToRabbitMQ()
	if err != nil {
		log.Panic("RabbitMQ 연결 실패: ", err)
	}
	defer mqClient.Conn.Close()

	// Redis 연결
	redisClient, err := redis.NewRedisClient()
	if err != nil {
		log.Panic("Redis 연결 실패: ", err)
	}
	defer redisClient.Client.Close()

	// 이벤트 발행자 (Emitter)
	emitter := event.NewEmitter(mqClient)

	// 서비스 계층
	chatRepo := repo.NewChatRepository(mongoClient) // Repository 생성
	gameService := service.NewGameService(redisClient, emitter, chatRepo)

	// WebSocket 핸들러
	gameHandler := handler.NewGameHandler(gameService)

	// 이벤트 소비자 (RabbitMQ Consumer)
	eventConsumer := event.NewConsumer(mqClient, redisClient, gameService)
	go eventConsumer.StartListening()

	// WebSocket 서버 실행
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: transport.NewRouter(gameHandler, redisClient),
	}

	log.Printf("🚀 Game Service Started on Port %d", webPort)
	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
