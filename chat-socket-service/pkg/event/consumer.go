package event

import (
	"encoding/json"
	"log"

	"github.com/baaami/dorandoran/chat-socket-service/pkg/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	conn *amqp.Connection
}

// NewConsumer 함수: RabbitMQ Consumer 초기화
func NewConsumer(conn *amqp.Connection) (*Consumer, error) {
	consumer := &Consumer{
		conn: conn,
	}

	err := consumer.setup()
	if err != nil {
		log.Printf("Failed to setup consumer: %v", err)
		return nil, err
	}

	return consumer, nil
}

func (consumer *Consumer) setup() error {
	channel, err := consumer.conn.Channel()
	if err != nil {
		return err
	}

	// Exchange 선언
	return declareExchange(channel)
}

// Listen 함수
func (c *Consumer) Listen(routingKeys []string, eventChannel chan<- types.WebSocketMessage) error {
	log.Println("Setting up listener for routing keys:", routingKeys)

	channel, err := c.conn.Channel()
	if err != nil {
		log.Printf("Failed to open channel: %v", err)
		return err
	}
	defer channel.Close()

	// Declare a unique, exclusive queue for this consumer
	queue, err := channel.QueueDeclare(
		"",    // Name: empty string for a random name
		false, // Durable: not persistent
		false, // Auto-delete: delete when unused
		true,  // Exclusive: this consumer only
		false, // No-wait
		nil,   // Arguments
	)
	if err != nil {
		log.Printf("Failed to declare queue: %v", err)
		return err
	}

	// Bind the queue to all provided routing keys
	for _, key := range routingKeys {
		err = channel.QueueBind(
			queue.Name,
			key,         // Routing key
			"app_topic", // Exchange name
			false,       // No-wait
			nil,         // Arguments
		)
		if err != nil {
			log.Printf("Failed to bind queue to routing key %s: %v", key, err)
			return err
		}
		log.Printf("Queue %s bound to routing key %s", queue.Name, key)
	}

	// Consume messages from the queue
	messages, err := channel.Consume(
		queue.Name, // Queue name
		"",         // Consumer tag
		true,       // Auto-acknowledge
		false,      // Exclusive
		false,      // No-local
		false,      // No-wait
		nil,        // Arguments
	)
	if err != nil {
		log.Printf("Failed to consume messages: %v", err)
		return err
	}

	// Start processing messages
	go func() {
		for d := range messages {
			log.Printf("Message received: %s", string(d.Body))

			// Unmarshal the message into the EventPayload struct
			var payload types.EventPayload
			err := json.Unmarshal(d.Body, &payload)
			if err != nil {
				log.Printf("Failed to unmarshal message: %v", err)
				continue
			}

			// Process the event payload based on its EventType
			if payload.EventType == "chat.latest" {
				var chatLatest types.ChatLatestEvent
				if err := json.Unmarshal(payload.Data, &chatLatest); err != nil {
					log.Printf("Failed to unmarshal chat.latest event: %v", err)
					continue
				}

				log.Printf("Send chat.latest event for RoomID: %s", chatLatest.RoomID)

				payload, err := json.Marshal(types.RoomJoinEvent{
					RoomID: chatLatest.RoomID,
				})
				if err != nil {
					log.Printf("Failed to marshal payload for chat.latest event: %v", err)
					continue
				}

				wsMessage := types.WebSocketMessage{
					Kind:    types.MessageKindChatLastest,
					Payload: json.RawMessage(payload),
				}

				eventChannel <- wsMessage
			}
		}
	}()

	log.Printf("Listening for messages on exchange 'app_topic' with routing keys: %v", routingKeys)
	select {} // Block forever
}
