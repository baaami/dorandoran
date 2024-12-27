package event

import (
	"encoding/json"

	"log"

	"github.com/baaami/dorandoran/chat/pkg/types"
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
	return declareMatchExchange(channel)
}

func (consumer *Consumer) Listen(chatRoomCreateChan chan<- types.MatchEvent) error {
	channel, err := consumer.conn.Channel()
	if err != nil {
		log.Printf("Failed to open channel: %v", err)
		return err
	}
	defer channel.Close()

	// Temporary queue 생성
	queue, err := channel.QueueDeclare(
		"match_queue", // name (empty for a temporary queue)
		false,         // durable
		true,          // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		log.Printf("Failed to declare queue: %v", err)
		return err
	}

	// Queue를 match_events exchange에 바인딩
	err = channel.QueueBind(
		queue.Name,     // queue name
		"",             // routing key (fanout type ignores this)
		"match_events", // exchange name
		false,          // no-wait
		nil,            // arguments
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

	log.Println("Listening for match_events...")

	// 메시지 처리 루프
	for msg := range messages {
		log.Printf("Received a message: %s", msg.Body)

		// 메시지를 MatchEvent로 파싱
		var matchEvent types.MatchEvent
		err := json.Unmarshal(msg.Body, &matchEvent)
		if err != nil {
			log.Printf("Failed to parse message as MatchEvent: %v", err)
			continue
		}

		// MatchEvent 로그 출력
		log.Printf("Parsed match event: MatchID=%s, MatchedUsers=%v", matchEvent.MatchId, matchEvent.MatchedUsers)

		// MatchEvent를 채널로 전송
		chatRoomCreateChan <- matchEvent
	}

	return nil
}
