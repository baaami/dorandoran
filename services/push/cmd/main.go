package main

import (
	"log"
	"solo/pkg/mq"
)

func main() {
	mqClient, err := mq.ConnectToRabbitMQ()
	if err != nil {
		log.Panic("RabbitMQ 연결 실패: ", err)
	}
	defer mqClient.Conn.Close()
}
