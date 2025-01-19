package event

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

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
