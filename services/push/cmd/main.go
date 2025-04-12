package main

import (
	"log"
	"solo/pkg/db"
	"solo/pkg/logger"
	"solo/pkg/mq"
	"solo/services/push/event"
	"solo/services/user/repository"
	"solo/services/user/service"
)

func main() {
	logger.InitLogger(logger.ServiceTypePush)

	dbConn, err := db.ConnectMySQL()
	if err != nil {
		log.Panic("MySQL 연결 실패: ", err)
	}

	mqClient, err := mq.ConnectToRabbitMQ()
	if err != nil {
		log.Panic("RabbitMQ 연결 실패: ", err)
	}
	defer mqClient.Conn.Close()

	userRepo := repository.NewUserRepository(dbConn)     // Repository 생성
	filterRepo := repository.NewFilterRepository(dbConn) // Repository 생성
	userService := service.NewUserService(userRepo, filterRepo)

	consumer := event.NewConsumer(mqClient, userService)
	consumer.StartListening()
}
