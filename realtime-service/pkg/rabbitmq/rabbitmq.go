package rabbitmq

import (
	"fmt"
	"log"
	"math"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQ 연결을 관리하는 구조체
type RabbitMQConnection struct {
	Conn *amqp.Connection
}

// RabbitMQ 연결 생성 함수
func NewRabbitMQConnection() (*RabbitMQConnection, error) {
	var counts int64
	var backOff = 1 * time.Second

	// RabbitMQ 연결 시도
	for {
		conn, err := amqp.Dial("amqp://guest:guest@rabbitmq")
		if err != nil {
			fmt.Println("RabbitMQ not yet ready, retrying...")
			counts++
		} else {
			log.Println("Connected to RabbitMQ!")
			return &RabbitMQConnection{Conn: conn}, nil
		}

		// 재시도 횟수가 5회를 넘으면 에러 반환
		if counts > 5 {
			return nil, fmt.Errorf("failed to connect to RabbitMQ after retries: %w", err)
		}

		backOff = time.Duration(math.Pow(float64(counts), 2)) * time.Second
		time.Sleep(backOff)
	}
}
