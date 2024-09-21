package match

import (
	"fmt"
	"log"

	"github.com/baaami/dorandoran/realtime/pkg/rabbitmq"
	socketio "github.com/googollee/go-socket.io"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Config struct {
	ws     *socketio.Server
	Rabbit *amqp.Connection
}

func RegisterMatchSocketServer(rabbitConn *rabbitmq.RabbitMQConnection) {
	app := Config{
		Rabbit: rabbitConn.Conn,
	}

	app.ws = socketio.NewServer(nil)

	app.ws.OnConnect("/match", func(s socketio.Conn) error {
		fmt.Println("connected to match id:", s.ID())
		return nil
	})

	app.ws.OnEvent("/match", "start", func(s socketio.Conn, matchID string) string {
		log.Printf("Match started for matchID: %s", matchID)

		// 예: RabbitMQ로 match event 푸시
		// emitter, err := event.NewEventEmitter(app.Rabbit)
		// if err != nil {
		// 	log.Printf("Failed to NewEventEmitter, err: %s", err.Error())
		// 	return err.Error()
		// }
		// emitter.PushMatchEventToQueue(matchID)

		s.Emit("match-started", "Match started successfully")
		return "Match started"
	})

	go func() {
		if err := app.ws.Serve(); err != nil {
			log.Fatalf("Socket.IO server error: %v", err)
		}
	}()
}
