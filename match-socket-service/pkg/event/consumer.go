package event

import (
	"encoding/json"
	"log"

	"github.com/baaami/dorandoran/match-socket-service/pkg/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	conn *amqp.Connection
}

func NewConsumer(conn *amqp.Connection) (Consumer, error) {
	consumer := Consumer{
		conn: conn,
	}

	err := consumer.setup()
	if err != nil {
		log.Printf("Failed to setup consumer: %v", err)
		return Consumer{}, err
	}

	return consumer, nil
}

func (consumer *Consumer) setup() error {
	channel, err := consumer.conn.Channel()
	if err != nil {
		return err
	}

	// Exchange 선언
	return declareChatRoomEventsExchange(channel)
}

func declareChatRoomEventsExchange(channel *amqp.Channel) error {
	exchange := "chat_room_create_events"
	return channel.ExchangeDeclare(
		exchange, // name
		"fanout", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
}

// Listen listens to chat_room_create_events exchange and processes events
func (consumer *Consumer) Listen(chatRoomChan chan<- types.ChatRoom) error {
	channel, err := consumer.conn.Channel()
	if err != nil {
		log.Printf("Failed to open channel: %v", err)
		return err
	}
	defer channel.Close()

	// Temporary queue 생성
	queue, err := channel.QueueDeclare(
		"",    // name (empty for a temporary queue)
		false, // durable
		true,  // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Printf("Failed to declare queue: %v", err)
		return err
	}

	// Queue를 chat_room_create_events exchange에 바인딩
	err = channel.QueueBind(
		queue.Name,                // queue name
		"",                        // routing key (fanout type ignores this)
		"chat_room_create_events", // exchange name
		false,                     // no-wait
		nil,                       // arguments
	)
	if err != nil {
		log.Printf("Failed to bind queue: %v", err)
		return err
	}

	// Queue에서 메시지 소비 시작
	messages, err := channel.Consume(
		queue.Name, // queue
		"",         // consumer
		true,       // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		log.Printf("Failed to start consuming messages: %v", err)
		return err
	}

	log.Println("Listening for chat_room_create_events...")

	// 메시지 처리 루프
	for msg := range messages {
		log.Printf("Received a message: %s", msg.Body)

		// 메시지를 ChatRoom으로 파싱
		var chatRoom types.ChatRoom
		err := json.Unmarshal(msg.Body, &chatRoom)
		if err != nil {
			log.Printf("Failed to parse message as ChatRoom: %v", err)
			continue
		}

		// ChatRoom 로그 출력
		log.Printf("Parsed chat room event: ID=%s, Users=%v, CreatedAt=%v", chatRoom.ID, chatRoom.Users, chatRoom.CreatedAt)

		// 채널로 ChatRoom 데이터 전송
		chatRoomChan <- chatRoom
	}

	return nil
}
