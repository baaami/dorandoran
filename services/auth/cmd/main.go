package main

import (
	"fmt"
	"log"
	"solo/pkg/db"
	"solo/pkg/redis"
	"solo/services/auth/handler"
	"solo/services/auth/repository"
	"solo/services/auth/service"
	"solo/services/auth/transport"
	user_repository "solo/services/user/repository"
	user_service "solo/services/user/service"

	"github.com/labstack/echo/v4"
)

const webPort = 80

type Config struct {
	RedisClient *redis.RedisClient
}

func main() {
	dbConn, err := db.ConnectMySQL()
	if err != nil {
		log.Panic("MySQL 연결 실패: ", err)
	}

	redisClient, err := redis.NewRedisClient()
	if err != nil {
		log.Fatalf("Failed to initialize Redis client: %v", err)
	}

	// 의존성 주입 (DI)
	userRepo := user_repository.NewUserRepository(dbConn) // Repository 생성
	err = userRepo.InitDB()
	if err != nil {
		log.Panic("Failed to User DB Migration: ", err)
	}

	filterRepo := user_repository.NewFilterRepository(dbConn)        // Repository 생성
	userService := user_service.NewUserService(userRepo, filterRepo) // Service 생성

	authRepo := repository.NewAuthRepository(redisClient)
	authService := service.NewAuthService(authRepo)
	authHandler := handler.NewAuthHandler(authService, userService)

	e := echo.New()

	transport.RegisterAuthRoutes(e, authHandler)

	log.Printf("Starting Auth Service on port %d", webPort)
	err = e.Start(fmt.Sprintf(":%d", webPort))
	if err != nil {
		log.Fatal(err)
	}
}
