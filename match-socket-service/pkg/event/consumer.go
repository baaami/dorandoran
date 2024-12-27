package event

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/baaami/dorandoran/match-socket-service/pkg/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Consumer 구조체 정의
type Consumer struct {
	conn      *amqp.Connection
	exchanges []string
}

// MessageHandler 타입 정의
type MessageHandler func(payload types.EventPayload, eventChannel chan<- types.ChatRoom)

// NewConsumer 함수: RabbitMQ Consumer 초기화
func NewConsumer(conn *amqp.Connection, exchanges []string) (*Consumer, error) {
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
			exchange, // Exchange 이름
			"fanout", // Exchange 타입
			true,     // Durable
			false,    // Auto-deleted
			false,    // Internal
			false,    // No-wait
			nil,      // Arguments
		)
		if err != nil {
			return fmt.Errorf("failed to declare exchange %s: %v", exchange, err)
		}

		log.Printf("Declare exchange: %s", exchange)
	}
	return nil
}

// Listen 함수: 메시지 소비 및 핸들러 호출
func (c *Consumer) Listen(handlers map[string]MessageHandler, eventChannel chan<- types.ChatRoom) error {
	channel, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %v", err)
	}
	defer channel.Close()

	// Temporary Queue 선언
	queue, err := channel.QueueDeclare(
		"chat_room_create_events_queue", // Name (임시 큐)
		false,                           // Durable
		false,                           // Auto-delete
		false,                           // Exclusive
		false,                           // No-wait
		nil,                             // Arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %v", err)
	}

	// Queue 바인딩
	for _, exchange := range c.exchanges {
		err := channel.QueueBind(
			queue.Name, // Queue 이름
			"",         // Routing Key (fanout은 무시)
			exchange,   // Exchange 이름
			false,      // No-wait
			nil,        // Arguments
		)
		if err != nil {
			return fmt.Errorf("failed to bind queue to exchange %s: %v", exchange, err)
		}

		log.Printf("Queue %s bound to exchange %s", queue.Name, "chat_room_create_events")
	}

	// 메시지 소비
	messages, err := channel.Consume(
		queue.Name, // Queue
		"",         // Consumer
		true,       // Auto-ack (수동 Ack)
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

			var payload types.EventPayload
			err := json.Unmarshal(msg.Body, &payload)
			if err != nil {
				log.Printf("Failed to parse message as EventPayload: %v", err)
				continue
			}

			if handler, exists := handlers[payload.EventType]; exists {
				handler(payload, eventChannel)
			} else {
				log.Printf("No handler found for event type: %s", payload.EventType)
			}
		}
	}()

	select {} // Block forever
}

func ChatRoomCreateHandler(payload types.EventPayload, eventChannel chan<- types.ChatRoom) {
	var chatRoom types.ChatRoom
	err := json.Unmarshal(payload.Data, &chatRoom)
	if err != nil {
		log.Printf("failed to unmarshal room.create event: %v", err)
		return
	}

	// 로그 출력
	log.Printf("Chat Room Created: ID=%s, Users=%v", chatRoom.ID, chatRoom.Users)

	eventChannel <- chatRoom
}
