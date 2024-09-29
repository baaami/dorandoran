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

// declareRoomExchange declares the exchange for chat messages
func declareRoomExchange(ch *amqp.Channel) error {
	return ch.ExchangeDeclare(
		"room_topic", // 새로운 chat exchange 이름
		"topic",      // topic type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
}
