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

// 현재 여기서는 chat.latest 라우팅 키를 위해서만 사용됨 -> 변경이 필요해보임
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
