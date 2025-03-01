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
	redisClient, err := redis.NewRedisClient()
	if err != nil {
		log.Fatalf("Failed to initialize Redis client: %v", err)
	}

	authRepo := repository.NewAuthRepository(redisClient)
	authService := service.NewAuthService(authRepo)
	authHandler := handler.NewAuthHandler(authService)

	e := echo.New()

	transport.RegisterAuthRoutes(e, authHandler)

	log.Printf("Starting Auth Service on port %d", webPort)
	err = e.Start(fmt.Sprintf(":%d", webPort))
	if err != nil {
		log.Fatal(err)
	}
}
