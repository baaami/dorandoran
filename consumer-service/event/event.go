package event

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

func declareExchange(ch *amqp.Channel) error {
	return ch.ExchangeDeclare(
		"logs_topic", // name
		"topic",      // type
		true,         // durable?
		false,        // auto-deleted?
		false,        // internal?
		false,        // no-wait?
		nil,          // arguements?
	)
}

func declareChatExchange(channel *amqp.Channel) error {
	return channel.ExchangeDeclare(
		"chat_topic", // chat_topic exchange
		"topic",      // topic type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
}

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
