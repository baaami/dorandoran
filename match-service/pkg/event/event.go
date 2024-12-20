package event

import (
	"encoding/json"
	"log"

	"github.com/baaami/dorandoran/match-service/pkg/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

type EventPayload struct {
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

type Emitter struct {
	connection *amqp.Connection
}

func NewEmitter(conn *amqp.Connection) (*Emitter, error) {
	emitter := &Emitter{
		connection: conn,
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Printf("Failed to open channel: %v", err)
		return nil, err
	}

	exchange := "match_events"
	err = ch.ExchangeDeclare(
		exchange, // name
		"fanout", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		log.Printf("Failed to declare exchange: %v", err)
		return nil, err
	}

	return emitter, nil
}

func (e *Emitter) PublishMatchEvent(event types.MatchEvent) error {
	channel, err := e.connection.Channel()
	if err != nil {
		log.Printf("Failed to open channel: %v", err)
		return err
	}
	defer channel.Close()

	eventBody, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal match event: %v", err)
		return err
	}

	exchange := "match_events"
	err = channel.Publish(
		exchange, // exchange
		"",       // routing key
		false,    // mandatory
		false,    // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        eventBody,
		},
	)
	if err != nil {
		log.Printf("Failed to publish match event: %v", err)
		return err
	}

	log.Printf("Published match event: %s", eventBody)
	return nil
}
