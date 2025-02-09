package main

import (
	"fmt"
	"log"
	"solo/pkg/redis"
	"solo/services/auth/handler"
	"solo/services/auth/repository"
	"solo/services/auth/service"
	"solo/services/auth/transport"

	"github.com/labstack/echo/v4"
)

const webPort = 80

type Config struct {
	RedisClient *redis.RedisClient
}

func main() {
	// Redis 연결
	redisClient, err := redis.NewRedisClient()
	if err != nil {
		log.Fatalf("Failed to initialize Redis client: %v", err)
	}

	// 레포지토리 및 서비스 초기화
	authRepo := repository.NewAuthRepository(redisClient)
	authService := service.NewAuthService(authRepo)
	authHandler := handler.NewAuthHandler(authService)

	// Echo 인스턴스 생성
	e := echo.New()

	// 라우팅 설정
	transport.RegisterAuthRoutes(e, authHandler)

	// HTTP 서버 설정 및 시작
	log.Printf("Starting Auth Service on port %d", webPort)
	err = e.Start(fmt.Sprintf(":%d", webPort))
	if err != nil {
		log.Fatal(err)
	}
}
