package main

import (
	"log"
	"os"

	"github.com/baaami/dorandoran/push/pkg/event"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Config struct {
	Rabbit *amqp.Connection
}

func main() {
	rabbitConn, err := connectToRabbitMQ()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer rabbitConn.Close()

	log.Printf("Starting push service")

	routingConfigs := []event.RoutingConfig{
		{
			Exchange: event.ExchangeConfig{Name: event.ExchangeAppTopic, Type: "topic"},
			Keys:     []string{event.EventTypeChat, event.EventTypeRoomTimeout},
		},
	}

	consumer, err := event.NewConsumer(rabbitConn, routingConfigs)
	if err != nil {
		log.Printf("Failed to make consumer: %v", err)
		os.Exit(1)
	}

	err = consumer.Listen()
	if err != nil {
		log.Println(err)
	}
}

func connectToRabbitMQ() (*amqp.Connection, error) {
	return amqp.Dial("amqp://guest:guest@rabbitmq")
}
