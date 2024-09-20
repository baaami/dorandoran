package event

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

// declareAuthExchange declares the exchange for login
func declareAuthExchange(channel *amqp.Channel) error {
	return channel.ExchangeDeclare(
		"auth_topic", // 새로운 auth exchange 이름
		"topic",      // topic type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
}

func declareRandomQueue(ch *amqp.Channel) (amqp.Queue, error) {
	return ch.QueueDeclare(
		"",    // name?
		false, // durable?
		false, // delete when unused?
		true,  // exclusive?
		false, // no-wait?
		nil,   // arguments?
	)
}
