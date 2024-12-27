package event

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/baaami/dorandoran/match-service/pkg/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Emitter 구조체
type Emitter struct {
	connection *amqp.Connection
	exchanges  map[string]ExchangeConfig
}

// NewEmitter 함수: Emitter 초기화
func NewEmitter(conn *amqp.Connection, exchanges []ExchangeConfig) (*Emitter, error) {
	emitter := &Emitter{
		connection: conn,
		exchanges:  make(map[string]ExchangeConfig),
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %v", err)
	}
	defer channel.Close()

	// Exchange 설정
	for _, exchange := range exchanges {
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
			return nil, fmt.Errorf("failed to declare exchange %s: %v", exchange.Name, err)
		}
		emitter.exchanges[exchange.Name] = exchange
		log.Printf("Declared exchange: %s", exchange.Name)
	}

	return emitter, nil
}

// publish 함수: 메시지 발행
func (e *Emitter) publish(exchangeName, routingKey string, payload EventPayload) error {
	channel, err := e.connection.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %v", err)
	}
	defer channel.Close()

	// EventPayload를 JSON으로 직렬화
	messageBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	// 메시지 발행
	err = channel.Publish(
		exchangeName,
		routingKey,
		false, // Mandatory
		false, // Immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        messageBody,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %v", err)
	}

	log.Printf("Message published to exchange %s with routing key %s: %s", exchangeName, routingKey, messageBody)
	return nil
}

// PublishMatchEvent 함수: match 이벤트 발행
func (e *Emitter) PublishMatchEvent(event types.MatchEvent) error {
	payload := EventPayload{
		EventType: "match",
		Data:      toJSON(event),
	}
	return e.publish("match_events", "", payload)
}

// toJSON 함수: 데이터를 JSON으로 변환
func toJSON(data interface{}) json.RawMessage {
	bytes, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal data: %v", err)
		return nil
	}
	return json.RawMessage(bytes)
}
