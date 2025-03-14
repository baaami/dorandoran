package main

import (
	"fmt"
	"log"
	"net/http"
	"solo/pkg/mq"
	"solo/pkg/redis"
	"solo/services/match/event"
	"solo/services/match/handler"
	"solo/services/match/service"
	"solo/services/match/transport"
)

const webPort = 80

func main() {
	mqClient, err := mq.ConnectToRabbitMQ()
	if err != nil {
		log.Panic("RabbitMQ 연결 실패: ", err)
	}
	defer mqClient.Conn.Close()

	redisClient, err := redis.NewRedisClient()
	if err != nil {
		log.Panic("Redis 연결 실패: ", err)
	}

	// Emitter 생성 (event 패키지 직접 참조 X)
	emitter := event.NewEmitter(mqClient)

	matchService := service.NewMatchService(redisClient, mqClient, emitter)
	matchHandler := handler.NewMatchHandler(matchService)

	consumer := event.NewConsumer(mqClient, matchService)
	consumer.StartListening()

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: transport.NewRouter(matchHandler, redisClient),
	}

	log.Printf("🚀 Match Service Started on Port %d", webPort)
	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
