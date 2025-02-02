package mq

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	Conn    *amqp.Connection
	channel *amqp.Channel
}

// ConnectToRabbitMQ: RabbitMQ 연결 설정
func ConnectToRabbitMQ() (*RabbitMQ, error) {
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq")
	if err != nil {
		log.Printf("❌ Failed to connect to RabbitMQ: %v", err)
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Printf("❌ Failed to open RabbitMQ channel: %v", err)
		return nil, err
	}

	return &RabbitMQ{Conn: conn, channel: ch}, nil
}

// DeclareExchange: Exchange 생성
func (mq *RabbitMQ) DeclareExchange(name, exchangeType string) error {
	return mq.channel.ExchangeDeclare(
		name,         // exchange name
		exchangeType, // type: topic or fanout
		true,         // durable
		false,        // autoDelete
		false,        // internal
		false,        // noWait
		nil,          // arguments
	)
}

// DeclareQueue: Queue 생성 및 바인딩
func (mq *RabbitMQ) DeclareQueue(queueName, exchangeName, routingKey string) (amqp.Queue, error) {
	queue, err := mq.channel.QueueDeclare(
		queueName, // queue name
		true,      // durable
		false,     // autoDelete
		false,     // exclusive
		false,     // noWait
		nil,       // arguments
	)
	if err != nil {
		return queue, err
	}

	// Exchange와 Queue 바인딩
	err = mq.channel.QueueBind(
		queue.Name,   // queue name
		routingKey,   // routing key
		exchangeName, // exchange name
		false,        // noWait
		nil,          // arguments
	)

	return queue, err
}

// PublishMessage: 메시지 발행
func (mq *RabbitMQ) PublishMessage(exchange, routingKey string, body []byte) error {
	return mq.channel.Publish(
		exchange,   // exchange name
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

// ConsumeMessages: 메시지 소비
func (mq *RabbitMQ) ConsumeMessages(queueName string, handler func([]byte)) error {
	msgs, err := mq.channel.Consume(
		queueName, // queue name
		"",        // consumer
		true,      // autoAck
		false,     // exclusive
		false,     // noLocal
		false,     // noWait
		nil,       // arguments
	)
	if err != nil {
		return err
	}

	// 메시지 처리
	go func() {
		for msg := range msgs {
			handler(msg.Body)
		}
	}()
	return nil
}
