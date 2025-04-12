package main

import (
	"fmt"
	"log"
	"net/http"
	"solo/pkg/db"
	"solo/pkg/logger"
	"solo/pkg/mq"
	"solo/services/user/event"
	"solo/services/user/handler"
	"solo/services/user/repository"
	"solo/services/user/service"
	"solo/services/user/transport"
)

const webPort = 80

func main() {
	logger.InitLogger(logger.ServiceTypeUser)

	dbConn, err := db.ConnectMySQL()
	if err != nil {
		log.Panic("MySQL 연결 실패: ", err)
	}

	mqClient, err := mq.ConnectToRabbitMQ()
	if err != nil {
		log.Panic("RabbitMQ 연결 실패: ", err)
	}
	defer mqClient.Conn.Close()

	// 의존성 주입 (DI)
	userRepo := repository.NewUserRepository(dbConn) // Repository 생성
	err = userRepo.InitDB()
	if err != nil {
		log.Panic("Failed to User DB Migration: ", err)
	}

	filterRepo := repository.NewFilterRepository(dbConn)     // Repository 생성
	filterService := service.NewFilterService(filterRepo)    // Service 생성
	filterHandler := handler.NewfilterHandler(filterService) // Handler 생성

	userService := service.NewUserService(userRepo, filterRepo) // Service 생성
	userHandler := handler.NewUserHandler(userService)          // Handler 생성

	consumer := event.NewConsumer(mqClient, userService)
	consumer.StartListening()

	router := transport.NewRouter(userHandler, filterHandler)

	log.Printf("🚀 User Service Started on Port %d", webPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", webPort), router))
}
