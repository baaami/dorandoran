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

	matchService := service.NewMatchService(redisClient, mqClient)
	matchHandler := handler.NewMatchHandler(matchService)

	consumer := event.NewConsumer(mqClient, matchService)
	consumer.StartListening()

	// Echo Router 설정
	router := transport.NewRouter(matchHandler)

	log.Printf("🚀 Match Service Started on Port %d", webPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", webPort), router))
}
