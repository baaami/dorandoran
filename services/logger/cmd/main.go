package main

import (
	"context"
	"log"
	"time"

	"solo/pkg/db"
	"solo/pkg/mq"
	"solo/services/logger/event"
	"solo/services/logger/repo"
)

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

	logRepo, err := repo.NewLogRepository(mongoClient)
	if err != nil {
		log.Panic("LogRepository 생성 실패: ", err)
	}

	eventConsumer := event.NewConsumer(mqClient, logRepo)
	go eventConsumer.StartListening()

	log.Println("🚀 Logger Service Started")
	select {}
}
