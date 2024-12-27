package main

import (
	"log"
	"os"
	"time"

	"github.com/baaami/dorandoran/match-service/pkg/event"
	"github.com/baaami/dorandoran/match-service/pkg/redis"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	// Redis 클라이언트 초기화
	redisClient, err := redis.NewRedisClient()
	if err != nil {
		log.Fatalf("Failed to initialize Redis client: %v", err)
		os.Exit(1)
	}

	// RabbitMQ 연결
	rabbitConn, err := connectToRabbitMQ()
	if err != nil {
		log.Fatalf("Failed to connect RAbbitMQ client: %v", err)
		os.Exit(1)
	}
	defer rabbitConn.Close()

	exchanges := []event.ExchangeConfig{
		{Name: "match_events", Type: "fanout"},
	}

	emitter, err := event.NewEmitter(rabbitConn, exchanges)
	if err != nil {
		log.Fatalf("Failed to initialize RabbitMQ emitter: %v", err)
	}

	log.Println("Starting match-service...")

	// 주기적으로 매칭 모니터링
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for coupleCount := 2; coupleCount <= 6; coupleCount++ {
				err := redisClient.MonitorAndMatchUsers(coupleCount, emitter)
				if err != nil {
					log.Printf("Error while monitoring queue for %d: %v", coupleCount, err)
				}
			}
		}
	}
}

func connectToRabbitMQ() (*amqp.Connection, error) {
	return amqp.Dial("amqp://guest:guest@rabbitmq")
}
