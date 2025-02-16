package main

import (
	"fmt"
	"log"
	"net/http"

	"solo/pkg/mq"
	"solo/pkg/redis"
	"solo/services/game/event"
	"solo/services/game/handler"
	"solo/services/game/service"
	"solo/services/game/transport"
)

const webPort = 80

func main() {

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
	gameService := service.NewGameService(redisClient, emitter)

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
