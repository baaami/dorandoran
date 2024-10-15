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

type RoomJoinEvent struct {
	RoomID string `bson:"room_id" json:"room_id"`
	UserID string `bson:"user_id" json:"user_id"`
}

type Chat struct {
	RoomID    string    `bson:"room_id" json:"room_id"`
	SenderID  string    `bson:"sender_id" json:"sender_id"`
	Message   string    `bson:"message" json:"message"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
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
		"app_topic",
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

func (e *Emitter) PushChatToQueue(chatMsg Chat) error {
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

	// EventPayload를 JSON으로 변환 (문자열로 변환하지 않음)
	eventJSON, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal event payload: %v", err)
		return err
	}

	// 메시지 발행 (eventJSON을 문자열이 아닌 바이트 슬라이스로 전송)
	err = e.PushBytes(eventJSON, "chat")
	if err != nil {
		log.Printf("Failed to push message to queue: %v", err)
		return err
	}

	log.Printf("Chat message successfully pushed to RabbitMQ")
	return nil
}

func (e *Emitter) PushRoomJoinToQueue(roomJoinMsg RoomJoinEvent) error {
	if e.connection == nil {
		log.Println("RabbitMQ connection is nil")
		return fmt.Errorf("RabbitMQ connection is nil")
	}

	// 채팅 확인 메시지 데이터를 JSON으로 변환
	roomJoinData, err := json.Marshal(roomJoinMsg)
	if err != nil {
		log.Printf("Failed to marshal room message: %v", err)
		return err
	}

	// EventPayload에 맞게 데이터를 래핑
	payload := EventPayload{
		EventType: "room.join",
		Data:      roomJoinData,
	}

	// EventPayload를 JSON으로 변환 (문자열로 변환하지 않음)
	eventJSON, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal event payload: %v", err)
		return err
	}

	// 메시지 발행 (eventJSON을 문자열이 아닌 바이트 슬라이스로 전송)
	err = e.PushBytes(eventJSON, "room.join")
	if err != nil {
		log.Printf("Failed to push message to queue: %v", err)
		return err
	}

	log.Printf("Room join event successfully pushed to RabbitMQ")
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
		"app_topic", // 교환기 이름
		severity,    // 라우팅 키
		false,       // mandatory
		false,       // immediate
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
