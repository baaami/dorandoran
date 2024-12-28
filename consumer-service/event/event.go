package event

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

func declareExchange(ch *amqp.Channel) error {
	return ch.ExchangeDeclare(
		"app_topic", // name
		"topic",     // type
		true,        // durable?
		false,       // auto-deleted?
		false,       // internal?
		false,       // no-wait?
		nil,         // arguements?
	)
}

func declareAppTopicQueue(ch *amqp.Channel) (amqp.Queue, error) {
	return ch.QueueDeclare(
		"app_topic_queue", // name?
		false,             // durable?
		false,             // delete when unused?
		true,              // exclusive?
		false,             // no-wait?
		nil,               // arguments?
	)
}
