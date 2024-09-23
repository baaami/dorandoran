// gateway-service/cmd/api/main.go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/baaami/dorandoran/broker/pkg/redis"
	"github.com/baaami/dorandoran/broker/pkg/socket"
	amqp "github.com/rabbitmq/amqp091-go"
)

const webPort = 80

type Config struct {
	Rabbit *amqp.Connection
}

func main() {
	// RabbitMQ 연결
	rabbitConn, err := connect()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer rabbitConn.Close()

	// Redis 연결
	redisClient, err := redis.NewRedisClient()
	if err != nil {
		log.Fatalf("Failed to initialize Redis client: %v", err)
	}

	app := Config{
		Rabbit: rabbitConn,
	}

	// WebSocket 설정
	wsConfig := &socket.Config{
		Clients:     sync.Map{},
		Rabbit:      rabbitConn,
		RedisClient: redisClient,
	}

	// Redis 대기열 모니터링 고루틴 실행
	go wsConfig.MonitorQueue()

	log.Printf("Starting Gateway service on port %d", webPort)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: app.routes(wsConfig),
	}

	err = srv.ListenAndServe()
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

func connect() (*amqp.Connection, error) {
	return amqp.Dial("amqp://guest:guest@rabbitmq")
}
