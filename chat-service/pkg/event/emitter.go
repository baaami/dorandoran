package event

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// EventPayload 구조체 정의
type EventPayload struct {
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

type ChatLatestEvent struct {
	RoomID string `json:"room_id"`
}

// Emitter 구조체 정의
type Emitter struct {
	connection *amqp.Connection
}

// NewEmitter 함수: RabbitMQ Emitter 초기화
func NewEmitter(conn *amqp.Connection) (*Emitter, error) {
	emitter := &Emitter{
		connection: conn,
	}

	channel, err := conn.Channel()
	if err != nil {
		log.Printf("Failed to open channel: %v", err)
		return nil, err
	}

	err = declareExchange(channel)
	if err != nil {
		log.Printf("Failed to declare exchange: %v", err)
		return nil, err
	}

	return emitter, nil
}

// PushChatLatestEvent 함수: WebSocket 알림 송신
func (e *Emitter) PushChatLatestEvent(chatLatest ChatLatestEvent) error {
	channel, err := e.connection.Channel()
	if err != nil {
		log.Printf("Failed to open channel: %v", err)
		return err
	}
	defer channel.Close()

	// WebSocketNotification을 JSON으로 직렬화
	data, err := json.Marshal(chatLatest)
	if err != nil {
		log.Printf("Failed to marshal WebSocket chatLatest data: %v", err)
		return err
	}

	// EventPayload 생성
	eventPayload := EventPayload{
		EventType: "chat.latest", // 채팅 데이터 최신화 (읽음 처리 완료)
		Data:      data,          // 알림 데이터
	}

	// EventPayload를 JSON으로 직렬화
	messageBody, err := json.Marshal(eventPayload)
	if err != nil {
		log.Printf("Failed to marshal EventPayload: %v", err)
		return err
	}

	// RabbitMQ 메시지 송신
	err = channel.Publish(
		"app_topic",   // exchange
		"chat.latest", // routing key
		false,         // mandatory
		false,         // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        messageBody,
		},
	)
	if err != nil {
		log.Printf("Failed to publish WebSocket chatLatest: %v", err)
		return err
	}

	log.Printf("WebSocket chatLatest published: %+v", chatLatest)
	return nil
}
