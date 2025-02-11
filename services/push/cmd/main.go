package main

import (
	"log"
	"solo/pkg/mq"
	"solo/services/push/event"
)

func main() {
	mqClient, err := mq.ConnectToRabbitMQ()
	if err != nil {
		log.Panic("RabbitMQ 연결 실패: ", err)
	}
	defer mqClient.Conn.Close()

	consumer := event.NewConsumer(mqClient)
	consumer.StartListening()
}
