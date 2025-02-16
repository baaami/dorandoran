package event

import (
	"log"
	"solo/pkg/mq"
)

type Consumer struct {
	mqClient     *mq.RabbitMQ
	eventHandler *EventHandler
}

func NewConsumer(mqClient *mq.RabbitMQ) *Consumer {
	return &Consumer{
		mqClient:     mqClient,
		eventHandler: NewEventHandler(),
	}
}

func (c *Consumer) StartListening() {
	// Exchange 및 Queue 설정
	err := c.mqClient.DeclareExchange(mq.ExchangeAppTopic, mq.ExchangeTypeTopic)
	if err != nil {
		log.Fatalf("❌ Failed to declare exchange %s: %v", mq.ExchangeAppTopic, err)
	}

	// Queue 생성 및 바인딩
	queue, err := c.mqClient.DeclareQueue(mq.QueuePush, mq.ExchangeAppTopic, []string{mq.RoutingKeyChat, mq.RoutingKeyRoomTimeout})
	if err != nil {
		log.Fatalf("❌ Failed to declare queue %s for %s: %v", mq.QueuePush, mq.ExchangeAppTopic, err)
	}

	// 이벤트 핸들러 등록
	handlers := mq.EventHandlerMap{
		mq.EventTypeChat:        c.eventHandler.HandleChatEvent,
		mq.EventTypeRoomTimeout: c.eventHandler.HandleRoomTimeoutEvent,
	}

	// 메시지 소비 시작
	c.mqClient.ConsumeMessages(queue.Name, handlers)

	log.Println("✅ RabbitMQ Consumer Listening...")

	select {}
}
