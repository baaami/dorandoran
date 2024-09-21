package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	amqp "github.com/rabbitmq/amqp091-go"
)

const webPort = 80

type Config struct {
	clients map[string]*websocket.Conn // 유저 ID와 WebSocket 연결을 매핑
	mu      sync.Mutex                 // 동시성 제어
	Rabbit  *amqp.Connection
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	// RabbitMQ 연결
	rabbitConn, err := connectRabbitMQ()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer rabbitConn.Close()

	app := Config{
		clients: make(map[string]*websocket.Conn),
		Rabbit:  rabbitConn,
	}

	// 웹 서버 시작
	log.Printf("Starting Gateway service on port %d", webPort)

	// HTTP 서버 설정
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: app.routes(),
	}

	// 서버 시작
	log.Printf("Starting Gateway Server on port %d", webPort)
	err = srv.ListenAndServe()
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

func connectRabbitMQ() (*amqp.Connection, error) {
	var backOff = 1 * time.Second
	var connection *amqp.Connection
	for attempts := 0; attempts < 5; attempts++ {
		c, err := amqp.Dial("amqp://guest:guest@rabbitmq")
		if err == nil {
			log.Println("Connected to RabbitMQ!")
			connection = c
			break
		}
		time.Sleep(backOff)
		backOff *= 2
	}

	if connection == nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ")
	}
	return connection, nil
}
