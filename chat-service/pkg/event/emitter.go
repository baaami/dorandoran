package event

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/baaami/dorandoran/chat/cmd/data"
	amqp "github.com/rabbitmq/amqp091-go"
)

type EventPayload struct {
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

type Emitter struct {
	connection *amqp.Connection
}

// 이후 확장될 경우 Parameter에 event type을 넘겨주도록 개선
// event type에 따라 적합한 exchange type을 선언
func (e *Emitter) setup() error {
	channel, err := e.connection.Channel()
	if err != nil {
		return err
	}

	defer channel.Close()
	return declareRoomExchange(channel)
}

// 이후 확장될 경우 Parameter에 event type을 넘겨주도록 개선
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

func (e *Emitter) PushRoomToQueue(room data.ChatRoom) error {
	if e.connection == nil {
		log.Println("RabbitMQ connection is nil")
		return fmt.Errorf("RabbitMQ connection is nil")
	}

	// 채팅 메시지 데이터를 JSON으로 변환
	roomData, err := json.Marshal(room)
	if err != nil {
		log.Printf("Failed to marshal chat message: %v", err)
		return err
	}

	// EventPayload에 맞게 데이터를 래핑
	payload := EventPayload{
		EventType: "room.deleted",
		Data:      roomData,
	}

	// EventPayload를 JSON으로 변환 (문자열로 변환하지 않음)
	eventJSON, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal event payload: %v", err)
		return err
	}

	// 메시지 발행 (eventJSON을 문자열이 아닌 바이트 슬라이스로 전송)
	err = e.PushBytes(eventJSON, "room")
	if err != nil {
		log.Printf("Failed to push message to queue: %v", err)
		return err
	}

	log.Printf("Chat message successfully pushed to RabbitMQ")
	return nil
}

// PushBytes 함수는 바이트 슬라이스 데이터를 RabbitMQ로 전송
func (e *Emitter) PushBytes(event []byte, severity string) error {
	channel, err := e.connection.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	log.Println("Pushing to channel")

	// 메시지 전송
	err = channel.Publish(
		"room_topic", // 교환기 이름
		severity,     // 라우팅 키
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "application/json", // 콘텐츠 타입 설정
			Body:        event,              // 바이트 슬라이스 데이터를 메시지로 전송
		},
	)
	if err != nil {
		log.Printf("Publish failed, err: %s", err.Error())
		return err
	}

	return nil
}
