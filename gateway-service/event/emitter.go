package event

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

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

// PushChatToQueue pushes a message into RabbitMQ
func (e *Emitter) PushChatToQueue(chatMsg ChatMessage) error {
	if e.connection == nil {
		log.Println("RabbitMQ connection is nil")
		return fmt.Errorf("RabbitMQ connection is nil")
	}

	emitter, err := NewEventEmitter(e.connection)
	if err != nil {
		return err
	}

	payload := ChatMessage{
		SenderID:   chatMsg.SenderID,
		ReceiverID: chatMsg.ReceiverID,
		RoomID:     chatMsg.RoomID,
		Message:    chatMsg.Message,
		CreatedAt:  chatMsg.CreatedAt,
	}

	j, _ := json.MarshalIndent(&payload, "", "\t")
	err = emitter.Push(string(j), "chat")
	if err != nil {
		log.Printf("Failed to push message to queue: %v", err)
		return err
	}
	return nil
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
