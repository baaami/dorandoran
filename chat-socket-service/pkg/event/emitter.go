package event

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/baaami/dorandoran/chat-socket-service/pkg/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Emitter struct {
	connection *amqp.Connection
	exchanges  map[string]ExchangeConfig
}

func (e *Emitter) setup() error {
	channel, err := e.connection.Channel()
	if err != nil {
		return err
	}

	defer channel.Close()
	return declareChatExchange(channel)
}

func NewEmitter(conn *amqp.Connection, exchanges []ExchangeConfig) (*Emitter, error) {
	emitter := &Emitter{
		connection: conn,
		exchanges:  make(map[string]ExchangeConfig),
	}

	// Exchange 설정
	channel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %v", err)
	}
	defer channel.Close()

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
func (e *Emitter) publish(exchangeName, routingKey string, payload types.EventPayload) error {
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

	// 메시지 전송
	err = channel.Publish(
		exchangeName, // Exchange
		routingKey,   // Routing Key
		false,        // Mandatory
		false,        // Immediate
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

// PushChatToQueue 함수: 채팅 이벤트 발행
func (e *Emitter) PushChatToQueue(chatEventMsg types.ChatEvent) error {
	payload := types.EventPayload{
		EventType: "chat",
		Data:      toJSON(chatEventMsg),
	}
	return e.publish("app_topic", "chat", payload)
}

// PushRoomJoinToQueue 함수: 방 참가 이벤트 발행
func (e *Emitter) PushRoomJoinToQueue(roomJoinMsg types.RoomJoinEvent) error {
	payload := types.EventPayload{
		EventType: "room.join",
		Data:      toJSON(roomJoinMsg),
	}
	return e.publish("app_topic", "room.join", payload)
}

func (e *Emitter) PushFinalChoiceTimeoutToQueue(finalChoiceTimeoutMsg types.FinalChoiceTimeoutEvent) error {
	payload := types.EventPayload{
		EventType: EventTypeFinalChoiceTimeout,
		Data:      toJSON(finalChoiceTimeoutMsg),
	}
	return e.publish("app_topic", EventTypeFinalChoiceTimeout, payload)
}

// Helper 함수: 데이터를 JSON 형식으로 변환
func toJSON(data interface{}) json.RawMessage {
	bytes, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal data: %v", err)
		return nil
	}
	return json.RawMessage(bytes)
}

func (e *Emitter) PublishMatchEvent(event types.MatchEvent) error {
	// EventPayload 생성
	payload := types.EventPayload{
		EventType: "match",
		Data:      toJSON(event), // MatchEvent 데이터를 JSON으로 직렬화
	}

	// 메시지 발행
	return e.publish("match_events", "", payload)
}
