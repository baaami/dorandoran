package event

import (
	"log"
	"solo/pkg/mq"
	"solo/services/match/service"
)

type Consumer struct {
	mqClient     *mq.RabbitMQ
	eventHandler *EventHandler
}

func NewConsumer(mqClient *mq.RabbitMQ, service *service.MatchService) *Consumer {
	return &Consumer{
		mqClient:     mqClient,
		eventHandler: NewEventHandler(service),
	}
}

func (c *Consumer) StartListening() {
	// Exchange 및 Queue 설정
	err := c.mqClient.DeclareExchange(mq.ExchangeChatRoomCreateEvents, mq.ExchangeTypeFanout)
	if err != nil {
		log.Fatalf("❌ Failed to declare exchange %s: %v", mq.ExchangeTypeFanout, err)
	}

	// Queue 생성 및 바인딩
	queue, err := c.mqClient.DeclareQueue(mq.QueueMatch, mq.ExchangeChatRoomCreateEvents, []string{})
	if err != nil {
		log.Fatalf("❌ Failed to declare queue %s for %s: %v", mq.QueueMatch, mq.ExchangeChatRoomCreateEvents, err)
	}

	// 이벤트 핸들러 등록
	handlers := mq.EventHandlerMap{
		mq.EventTypeRoomCreate: c.eventHandler.HandleRoomCreateEvent,
	}

	// 메시지 소비 시작
	c.mqClient.ConsumeMessages(queue.Name, handlers)

	log.Println("✅ RabbitMQ Consumer Listening...")
}
