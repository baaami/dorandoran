package event

import (
	"encoding/json"
	"fmt"

	"log"

	"github.com/baaami/dorandoran/chat/pkg/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

// MessageHandler 타입 정의
type MessageHandler func(payload EventPayload, eventChannel chan<- types.MatchEvent) error

type Consumer struct {
	conn      *amqp.Connection
	exchanges []ExchangeConfig
}

// NewConsumer 함수: RabbitMQ Consumer 초기화
func NewConsumer(conn *amqp.Connection, exchanges []ExchangeConfig) (*Consumer, error) {
	consumer := &Consumer{
		conn:      conn,
		exchanges: exchanges,
	}

	err := consumer.setup()
	if err != nil {
		log.Printf("Failed to setup consumer: %v", err)
		return nil, err
	}

	return consumer, nil
}

// setup 함수: 모든 Exchange 선언
func (c *Consumer) setup() error {
	channel, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %v", err)
	}
	defer channel.Close()

	for _, exchange := range c.exchanges {
		err := channel.ExchangeDeclare(
			exchange.Name,
			exchange.Type,
			true,  // Durable
			false, // Auto-deleted
			false, // Internal
			false, // No-wait
			nil,   // Arguments
		)
		if err != nil {
			return fmt.Errorf("failed to declare exchange %s: %v", exchange.Name, err)
		}
		log.Printf("Declared exchange: %s", exchange.Name)
	}
	return nil
}

// Listen 함수: 메시지 소비 및 핸들러 호출
func (c *Consumer) Listen(handlers map[string]MessageHandler, eventChannel chan<- types.MatchEvent) error {
	channel, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %v", err)
	}
	defer channel.Close()

	// Queue 선언
	queue, err := channel.QueueDeclare(
		"match_queue", // Queue 이름
		true,          // Durable (영구적)
		false,         // Auto-delete
		false,         // Exclusive
		false,         // No-wait
		nil,           // Arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %v", err)
	}

	// Queue와 Exchange 바인딩
	for _, exchange := range c.exchanges {
		err := channel.QueueBind(
			queue.Name, // Queue 이름
			"",         // Routing Key (fanout은 무시)
			exchange.Name,
			false, // No-wait
			nil,   // Arguments
		)
		if err != nil {
			return fmt.Errorf("failed to bind queue to exchange %s: %v", exchange.Name, err)
		}
		log.Printf("Queue %s bound to exchange %s", queue.Name, exchange.Name)
	}

	// 메시지 소비
	messages, err := channel.Consume(
		queue.Name, // Queue
		"",         // Consumer
		true,       // Auto-ack
		false,      // Exclusive
		false,      // No-local
		false,      // No-wait
		nil,        // Args
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming messages: %v", err)
	}

	log.Printf("Listening for messages on exchanges: %v", c.exchanges)

	// 메시지 처리
	go func() {
		for msg := range messages {
			log.Printf("Received a message: %s", string(msg.Body))

			// EventPayload 파싱
			var payload EventPayload
			err := json.Unmarshal(msg.Body, &payload)
			if err != nil {
				log.Printf("Failed to parse message as EventPayload: %v", err)
				continue
			}

			// 핸들러 호출
			if handler, exists := handlers[payload.EventType]; exists {
				if err := handler(payload, eventChannel); err != nil {
					log.Printf("Error handling event %s: %v", payload.EventType, err)
				}
			} else {
				log.Printf("No handler found for event type: %s", payload.EventType)
			}
		}
	}()

	select {} // Block forever
}

// Match 이벤트 핸들러
func MatchEventHandler(payload EventPayload, eventChannel chan<- types.MatchEvent) error {
	var matchEvent types.MatchEvent
	err := json.Unmarshal(payload.Data, &matchEvent)
	if err != nil {
		return fmt.Errorf("failed to unmarshal MatchEvent: %v", err)
	}

	log.Printf("Processed MatchEvent: MatchID=%s, MatchedUsers=%v", matchEvent.MatchId, matchEvent.MatchedUsers)
	eventChannel <- matchEvent
	return nil
}
