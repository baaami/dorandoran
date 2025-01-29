package event

import (
	"encoding/json"
	"fmt"

	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// MessageHandler 타입 정의
type MessageHandler func(payload EventPayload, eventChannel chan<- EventPayload) error

type Consumer struct {
	conn           *amqp.Connection
	routingConfigs []RoutingConfig
}

// NewConsumer 함수: RabbitMQ Consumer 초기화
func NewConsumer(conn *amqp.Connection, routingConfigs []RoutingConfig) (*Consumer, error) {
	consumer := &Consumer{
		conn:           conn,
		routingConfigs: routingConfigs,
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

	// 모든 Exchanges 선언
	for _, config := range c.routingConfigs {
		err := DeclareExchange(channel, config.Exchange)
		if err != nil {
			log.Printf("Failed to declare exchange %s: %v", config.Exchange.Name, err)
			return err
		}
	}
	return nil
}

// Listen 함수: 메시지 소비 및 핸들러 호출
func (c *Consumer) Listen(handlers map[string]MessageHandler, eventChannel chan<- EventPayload) error {
	channel, err := c.conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	for _, config := range c.routingConfigs {
		queue, err := channel.QueueDeclare("chat_queue", false, false, true, false, nil)
		if err != nil {
			return err
		}

		// topic exchnage
		if config.Exchange.Type == "topic" {
			for _, key := range config.Keys {
				err := channel.QueueBind(queue.Name, key, config.Exchange.Name, false, nil)
				if err != nil {
					log.Printf("Failed to bind queue %s to routing key %s: %v", queue.Name, key, err)
					return err
				}
			}
		} else if config.Exchange.Type == "fanout" {
			// fanout exchnage
			err := channel.QueueBind(queue.Name, "", config.Exchange.Name, false, nil)
			if err != nil {
				log.Printf("Failed to bind queue %s to routing key %s: %v", queue.Name, "", err)
				return err
			}
		}

		messages, err := channel.Consume(queue.Name, "", true, false, false, false, nil)
		if err != nil {
			return err
		}

		go func() {
			for d := range messages {
				var payload EventPayload
				if err := json.Unmarshal(d.Body, &payload); err != nil {
					log.Printf("Failed to unmarshal message: %v", err)
					continue
				}

				log.Printf("Received Event Type: %s", payload.EventType)

				if handler, ok := handlers[payload.EventType]; ok {
					handler(payload, eventChannel)
				} else {
					log.Printf("No handler for event type: %s", payload.EventType)
				}
			}
		}()
	}
	select {} // Block forever
}

// Match 이벤트 핸들러
func MatchEventHandler(payload EventPayload, eventChannel chan<- EventPayload) error {
	eventChannel <- payload
	return nil
}

func RoomTimeoutHandler(payload EventPayload, eventChannel chan<- EventPayload) error {
	eventChannel <- payload
	return nil
}

func FinalChoiceTimeoutHandler(payload EventPayload, eventChannel chan<- EventPayload) error {
	eventChannel <- payload
	return nil
}
