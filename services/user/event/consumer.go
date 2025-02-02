package event

import (
	"log"
	"solo/pkg/mq"
	"solo/services/user/service"
)

type Consumer struct {
	mqClient     *mq.RabbitMQ
	eventHandler *EventHandler
}

func NewConsumer(mqClient *mq.RabbitMQ, userService *service.UserService) *Consumer {
	return &Consumer{
		mqClient:     mqClient,
		eventHandler: NewEventHandler(userService),
	}
}

func (c *Consumer) StartListening() {
	// Exchange 및 Queue 설정
	err := c.mqClient.DeclareExchange(mq.ExchangeAppTopic, mq.ExchangeTypeTopic)
	if err != nil {
		log.Fatalf("❌ Failed to declare exchange %s: %v", mq.ExchangeAppTopic, err)
	}

	err = c.mqClient.DeclareExchange(mq.ExchangeMatch, mq.ExchangeTypeFanout)
	if err != nil {
		log.Fatalf("❌ Failed to declare exchange %s: %v", mq.ExchangeMatch, err)
	}

	// Queue 생성 및 바인딩
	queue, err := c.mqClient.DeclareQueue(mq.QueueUser, mq.ExchangeAppTopic, mq.RoutingKeyFinalChoiceTimeout)
	if err != nil {
		log.Fatalf("❌ Failed to declare queue %s for %s: %v", mq.QueueUser, mq.ExchangeAppTopic, err)
	}

	_, err = c.mqClient.DeclareQueue(mq.QueueUser, mq.ExchangeMatch, "")
	if err != nil {
		log.Fatalf("❌ Failed to declare queue %s for %s: %v", mq.QueueUser, mq.ExchangeMatch, err)
	}

	// 이벤트 리스닝 시작
	go c.mqClient.ConsumeMessages(queue.Name, c.eventHandler.HandleFinalChoiceTimeout)
	go c.mqClient.ConsumeMessages(queue.Name, c.eventHandler.HandleMatchEvent)
	go c.mqClient.ConsumeMessages(queue.Name, c.eventHandler.HandleRoomLeave)

	log.Println("✅ RabbitMQ Consumer Listening...")
}
