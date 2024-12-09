package event

import (
	"encoding/json"
	"log"

	"github.com/baaami/dorandoran/broker/pkg/data"
	amqp "github.com/rabbitmq/amqp091-go"
)

// ChatLatestEvent 정의
type ChatLatestEvent struct {
	RoomID string `json:"room_id"`
}

type RoomRemainingEvent struct {
	RoomID    string `json:"room_id"`
	Remaining int    `json:"remaining"` // 남은 시간 (초)
}

type RoomTimeoutEvent struct {
	RoomID string `json:"room_id"`
}

// Consumer 구조체 정의
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
func (c *Consumer) Listen(routingKeys []string, eventChannel chan<- data.WebSocketMessage) error {
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
			var payload EventPayload
			err := json.Unmarshal(d.Body, &payload)
			if err != nil {
				log.Printf("Failed to unmarshal message: %v", err)
				continue
			}

			// Process the event payload based on its EventType
			if payload.EventType == "chat.latest" {
				var chatLatest ChatLatestEvent
				if err := json.Unmarshal(payload.Data, &chatLatest); err != nil {
					log.Printf("Failed to unmarshal chat.latest event: %v", err)
					continue
				}

				log.Printf("Send chat.latest event for RoomID: %s", chatLatest.RoomID)

				payload, err := json.Marshal(RoomJoinEvent{
					RoomID: chatLatest.RoomID,
				})
				if err != nil {
					log.Printf("Failed to marshal payload for chat.latest event: %v", err)
					continue
				}

				wsMessage := data.WebSocketMessage{
					Kind:    data.MessageKindChatLastest,
					Payload: json.RawMessage(payload),
				}

				eventChannel <- wsMessage
			} else if payload.EventType == "room.remain.time" {
				var roomRemaining RoomRemainingEvent
				if err := json.Unmarshal(payload.Data, &roomRemaining); err != nil {
					log.Printf("Failed to unmarshal room.remain.time event: %v", err)
					continue
				}

				log.Printf("Send room_remaining event for RoomID: %s, time %d", roomRemaining.RoomID, roomRemaining.Remaining)

				payload, err := json.Marshal(RoomRemainingEvent{
					RoomID:    roomRemaining.RoomID,
					Remaining: roomRemaining.Remaining,
				})
				if err != nil {
					log.Printf("Failed to marshal payload for chat.latest event: %v", err)
					continue
				}

				wsMessage := data.WebSocketMessage{
					Kind:    data.MessageKindRoomRemaining,
					Payload: json.RawMessage(payload),
				}

				eventChannel <- wsMessage
			} else if payload.EventType == "room.timeout" {
				var roomTimeout RoomTimeoutEvent
				if err := json.Unmarshal(payload.Data, &roomTimeout); err != nil {
					log.Printf("Failed to unmarshal room.timeout event: %v", err)
					continue
				}

				log.Printf("Send room_timeout event for RoomID: %s", roomTimeout.RoomID)

				payload, err := json.Marshal(RoomTimeoutEvent{
					RoomID: roomTimeout.RoomID,
				})
				if err != nil {
					log.Printf("Failed to marshal payload for room.timeout event: %v", err)
					continue
				}

				wsMessage := data.WebSocketMessage{
					Kind:    data.MessageKindRoomTimeout,
					Payload: json.RawMessage(payload),
				}

				eventChannel <- wsMessage // 채널로 이벤트 전달
			}
		}
	}()

	log.Printf("Listening for messages on exchange 'app_topic' with routing keys: %v", routingKeys)
	select {} // Block forever
}

// processEvent handles events based on their type
func (c *Consumer) processEvent(payload EventPayload) {
	log.Printf("Processing event: %s", payload.EventType)

	switch payload.EventType {
	case "chat.latest":
		log.Printf("Handling chat.latest event: %s", string(payload.Data))
		var chatLatest ChatLatestEvent
		if err := json.Unmarshal(payload.Data, &chatLatest); err != nil {
			log.Printf("Failed to unmarshal chat.latest event: %v", err)
			return
		}

		handleChatLatestEvent(chatLatest)
	default:
		log.Printf("Unhandled event type: %s", payload.EventType)
	}
}

// handleChatLatestEvent 함수: chat.latest 이벤트 처리
func handleChatLatestEvent(chatLatest ChatLatestEvent) {
	log.Printf("Handling chat.latest event for RoomID: %s", chatLatest.RoomID)

	// WebSocket 클라이언트에게 알림 전송 (예: WebSocket 메시지 브로드캐스팅)
	// WebSocket 브로드캐스팅 구현이 필요함.
}
