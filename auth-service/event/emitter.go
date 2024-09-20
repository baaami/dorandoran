package event

import (
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type EventPayload struct {
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Nickname string `json:"nickname"`
	Gender   int    `json:"gender"`
	Age      int    `json:"age"`
	Email    string `json:"email"`
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
	return declareAuthExchange(channel)
}

func (e *Emitter) Push(event string, severity string) error {
	channel, err := e.connection.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	log.Println("Pushing to channel")

	err = channel.Publish(
		"auth_topic",
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

func (e *Emitter) PushUserToQueue(userMsg User) error {
	if e.connection == nil {
		log.Println("RabbitMQ connection is nil")
		return fmt.Errorf("RabbitMQ connection is nil")
	}

	// 유저 데이터를 JSON으로 변환
	userData, err := json.Marshal(userMsg)
	if err != nil {
		log.Printf("Failed to marshal user message: %v", err)
		return err
	}

	// EventPayload에 맞게 데이터를 래핑
	payload := EventPayload{
		EventType: "user.created",
		Data:      userData,
	}

	// JSON으로 변환
	eventJSON, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal event payload: %v", err)
		return err
	}

	// 메시지 발행
	err = e.Push(string(eventJSON), "user.created")
	if err != nil {
		log.Printf("Failed to push message to queue: %v", err)
		return err
	}

	log.Println("User creation event successfully pushed to RabbitMQ")
	return nil
}
