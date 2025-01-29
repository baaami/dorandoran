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

func declareMatchExchange(ch *amqp.Channel) error {
	exchange := "match_events"
	return ch.ExchangeDeclare(
		exchange, // name
		"fanout", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
}

func DeclareExchange(channel *amqp.Channel, exchangeConfig ExchangeConfig) error {
	return channel.ExchangeDeclare(
		exchangeConfig.Name, // Exchange 이름
		exchangeConfig.Type, // Exchange 타입 (topic, fanout 등)
		true,                // Durable
		false,               // Auto-deleted
		false,               // Internal
		false,               // No-wait
		nil,                 // Arguments
	)
}
