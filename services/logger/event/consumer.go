package event

import (
	"log"
	"solo/pkg/mq"
	eventtypes "solo/pkg/types/eventtype"
	"solo/services/logger/repo"
)

type Consumer struct {
	mqClient     *mq.RabbitMQ
	eventHandler *EventHandler
}

func NewConsumer(mqClient *mq.RabbitMQ, logRepo *repo.LogRepository) *Consumer {
	return &Consumer{
		mqClient:     mqClient,
		eventHandler: NewEventHandler(logRepo),
	}
}

func (c *Consumer) StartListening() {
	// Exchange 설정
	err := c.mqClient.DeclareExchange(mq.ExchangeLog, mq.ExchangeTypeFanout)
	if err != nil {
		log.Fatalf("❌ Failed to declare exchange %s: %v", mq.ExchangeLog, err)
	}

	// Queue 생성 및 바인딩
	queue, err := c.mqClient.DeclareQueue(mq.QueueLog, mq.ExchangeLog, []string{""})
	if err != nil {
		log.Fatalf("❌ Failed to declare queue %s for %s: %v", mq.QueueLog, mq.ExchangeLog, err)
	}

	// 이벤트 핸들러 등록
	handlers := mq.EventHandlerMap{
		eventtypes.EventTypeLog: c.eventHandler.HandleLogEvent,
	}

	// 메시지 소비 시작
	c.mqClient.ConsumeMessages(queue.Name, handlers)

	log.Println("✅ Logger Service Consumer Listening...")
}
