// gateway-service/cmd/api/main.go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

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

	app := Config{
		Rabbit: rabbitConn,
	}

	// WebSocket 서버 설정
	// WebSocket 설정
	wsConfig := &socket.Config{
		Clients: sync.Map{},
		Rabbit:  rabbitConn,
	}

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
