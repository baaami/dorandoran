package event

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

// declareChatExchange declares the exchange for chat messages
func declareChatExchange(ch *amqp.Channel) error {
	return ch.ExchangeDeclare(
		"chat_topic", // 새로운 chat exchange 이름
		"topic",      // topic type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
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
