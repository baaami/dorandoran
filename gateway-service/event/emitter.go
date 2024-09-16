package event

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type EventPayload struct {
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

type ChatMessage struct {
	RoomID     string    `bson:"room_id"`
	SenderID   string    `bson:"sender_id"`
	ReceiverID string    `bson:"receiver_id"`
	Message    string    `bson:"message"`
	CreatedAt  time.Time `bson:"created_at"`
}

type Emitter struct {
	connection *amqp.Connection
}

func (e *Emitter) setup() error {
	channel, err := e.connection.Channel()
	if err != nil {
		return err
	}

	defer channel.Close()
	return declareChatExchange(channel)
}

func (e *Emitter) Push(event string, severity string) error {
	channel, err := e.connection.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	log.Println("Pushing to channel")

	err = channel.Publish(
		"chat_topic",
		severity,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(event),
		},
	)
	if err != nil {
		log.Printf("Publish fail, err: %s", err.Error())
		return err
	}

	return nil
}

func NewEventEmitter(conn *amqp.Connection) (Emitter, error) {
	emitter := Emitter{
		connection: conn,
	}

	err := emitter.setup()
	if err != nil {
		return Emitter{}, err
	}

	return emitter, nil
}

func (e *Emitter) PushChatMessageToQueue(chatMsg ChatMessage) error {
	if e.connection == nil {
		log.Println("RabbitMQ connection is nil")
		return fmt.Errorf("RabbitMQ connection is nil")
	}

	// 채팅 메시지 데이터를 JSON으로 변환
	chatData, err := json.Marshal(chatMsg)
	if err != nil {
		log.Printf("Failed to marshal chat message: %v", err)
		return err
	}

	// EventPayload에 맞게 데이터를 래핑
	payload := EventPayload{
		EventType: "chat",
		Data:      chatData,
	}

	// JSON으로 변환
	eventJSON, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal event payload: %v", err)
		return err
	}

	// 메시지 발행
	err = e.Push(string(eventJSON), "chat")
	if err != nil {
		log.Printf("Failed to push message to queue: %v", err)
		return err
	}

	log.Printf("Chat message successfully pushed to RabbitMQ")
	return nil
}
