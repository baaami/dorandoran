package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/baaami/dorandoran/realtime/pkg/chat"
	"github.com/baaami/dorandoran/realtime/pkg/rabbitmq"
	"github.com/baaami/dorandoran/realtime/pkg/routes"
	socketio "github.com/googollee/go-socket.io"
	amqp "github.com/rabbitmq/amqp091-go"
)

const webPort = 80

type Config struct {
	Rabbit *amqp.Connection
}

func main() {
	// RabbitMQ 연결 생성
	rabbitConn, err := rabbitmq.NewRabbitMQConnection()
	if err != nil {
		log.Fatalf("Could not connect to RabbitMQ: %v", err)
	}
	defer rabbitConn.Conn.Close()

	// Socket.IO 서버 등록
	socketServer := socketio.NewServer(nil)
	chat.RegisterChatSocketServer(rabbitConn, socketServer)
	// match.RegisterMatchSocketServer(rabbitConn)

	// HTTP 서버 설정
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: routes.InitRoutes(socketServer),
	}

	// 서버 시작
	log.Printf("Starting Realtime service on port %d", webPort)
	err = srv.ListenAndServe()
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
