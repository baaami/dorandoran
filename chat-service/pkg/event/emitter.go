package event

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/baaami/dorandoran/chat/pkg/data"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Emitter 구조체 정의
type Emitter struct {
	connection *amqp.Connection
	exchanges  map[string]ExchangeConfig
}

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

	// Declare all exchanges
	for _, ex := range exchanges {
		err := channel.ExchangeDeclare(
			ex.Name,
			ex.Type,
			true,  // Durable
			false, // Auto-deleted
			false, // Internal
			false, // No-wait
			nil,   // Arguments
		)
		if err != nil {
			return nil, fmt.Errorf("failed to declare exchange %s: %v", ex.Name, err)
		}
		emitter.exchanges[ex.Name] = ex
	}

	return emitter, nil
}

func (e *Emitter) publish(exchangeName, routingKey string, payload EventPayload) error {
	channel, err := e.connection.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %v", err)
	}
	defer channel.Close()

	// Serialize payload
	messageBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal EventPayload: %v", err)
	}

	// Publish message
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
		return fmt.Errorf("failed to publish event: %v", err)
	}

	log.Printf("Event published to exchange %s with routing key %s", exchangeName, routingKey)
	return nil
}

// PublishChatRoomEvent publishes a chat room creation event
func (e *Emitter) PublishChatRoomCreateEvent(chatRoom data.ChatRoom) error {
	payload := EventPayload{
		EventType: EventTypeRoomCreate,
		Data:      toJSON(chatRoom),
	}
	return e.publish(ExchangeChatRoomCreateEvents, "", payload)
}

func (e *Emitter) PublishCoupleRoomCreateEvent(chatRoom data.ChatRoom) error {
	payload := EventPayload{
		EventType: EventTypeCoupleRoomCreate,
		Data:      toJSON(chatRoom),
	}
	return e.publish(ExchangeCoupleRoomCreateEvents, "", payload)
}

// 채팅 내용 최신화 필요 이벤트 발행
func (e *Emitter) PushChatLatestEvent(chatLatest ChatLatestEvent) error {
	payload := EventPayload{
		EventType: EventTypeChatLatest,
		Data:      toJSON(chatLatest),
	}
	return e.publish(ExchangeAppTopic, "chat.latest", payload)
}

// 채팅방에 유저 나감 이벤트 발행
func (e *Emitter) PushRoomLeaveEvent(roomLeave RoomLeaveEvent) error {
	payload := EventPayload{
		EventType: EventTypeRoomLeave,
		Data:      toJSON(roomLeave),
	}
	return e.publish(ExchangeAppTopic, "room.leave", payload)
}

// 채팅방 남은시간 이벤트 발행
func (e *Emitter) PushRoomRemainTime(roomID string, remaining int) error {
	roomRemainTime := data.RoomRemainingEvent{
		RoomID:    roomID,
		Remaining: remaining,
	}
	payload := EventPayload{
		EventType: EventTypeRoomRemainTime,
		Data:      toJSON(roomRemainTime),
	}
	return e.publish(ExchangeAppTopic, "room.remain.time", payload)
}

// 채팅방 타임아웃 이벤트 발행
func (e *Emitter) PushRoomTimeout(timeoutEvent RoomTimeoutEvent) error {
	payload := EventPayload{
		EventType: EventTypeRoomTimeout,
		Data:      toJSON(timeoutEvent),
	}
	return e.publish(ExchangeAppTopic, "room.timeout", payload)
}
