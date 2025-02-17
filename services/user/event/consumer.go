package event

import (
	"log"
	"solo/pkg/mq"
	eventtypes "solo/pkg/types/eventtype"
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

	err = c.mqClient.DeclareExchange(mq.ExchangeMatchEvents, mq.ExchangeTypeFanout)
	if err != nil {
		log.Fatalf("❌ Failed to declare exchange %s: %v", mq.ExchangeMatchEvents, err)
	}

	// Queue 생성 및 바인딩
	queue, err := c.mqClient.DeclareQueue(mq.QueueUser, mq.ExchangeAppTopic, []string{mq.RoutingKeyFinalChoiceTimeout})
	if err != nil {
		log.Fatalf("❌ Failed to declare queue %s for %s: %v", mq.QueueUser, mq.ExchangeAppTopic, err)
	}

	_, err = c.mqClient.DeclareQueue(mq.QueueUser, mq.ExchangeMatchEvents, []string{})
	if err != nil {
		log.Fatalf("❌ Failed to declare queue %s for %s: %v", mq.QueueUser, mq.ExchangeMatchEvents, err)
	}

	// 이벤트 핸들러 등록
	handlers := mq.EventHandlerMap{
		eventtypes.EventTypeMatch:              c.eventHandler.HandleMatchEvent,
		eventtypes.EventTypeFinalChoiceTimeout: c.eventHandler.HandleFinalChoiceTimeout,
		eventtypes.EventTypeRoomLeave:          c.eventHandler.HandleRoomLeave,
	}

	// 메시지 소비 시작
	c.mqClient.ConsumeMessages(queue.Name, handlers)

	log.Println("✅ RabbitMQ Consumer Listening...")
}
