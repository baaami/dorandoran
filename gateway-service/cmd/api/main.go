// gateway-service/cmd/api/main.go
package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/baaami/dorandoran/broker/pkg/redis"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

const webPort = 80

type Config struct{}

func main() {
	// RabbitMQ 연결
	rabbitConn, err := connect()
	if err != nil {
		log.Error().Msg(err.Error())
		os.Exit(1)
	}
	defer rabbitConn.Close()

	// Redis 연결
	redisClient, err := redis.NewRedisClient()
	if err != nil {
		log.Error().Msgf("Failed to initialize Redis client: %v", err)
	}

	app := Config{}

	log.Info().Msgf("Starting Gateway service on port %d", webPort)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: app.routes(redisClient),
	}

	err = srv.ListenAndServe()
	if err != nil {
		log.Error().Msgf("Error starting server: %v", err)
	}
}

func connect() (*amqp.Connection, error) {
	return amqp.Dial("amqp://guest:guest@rabbitmq")
}
