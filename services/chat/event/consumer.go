package event

import (
	"log"
	"solo/pkg/mq"
	"solo/pkg/redis"
	"solo/services/chat/service"
)

type Consumer struct {
	mqClient     *mq.RabbitMQ
	eventHandler *EventHandler
}

func NewConsumer(mqClient *mq.RabbitMQ, redisClient *redis.RedisClient, chatService *service.ChatService) *Consumer {
	return &Consumer{
		mqClient:     mqClient,
		eventHandler: NewEventHandler(chatService, redisClient),
	}
}

func (c *Consumer) StartListening() {
	err := c.mqClient.DeclareExchange(mq.ExchangeChatRoomCreateEvents, mq.ExchangeTypeFanout)
	if err != nil {
		log.Fatalf("❌ Failed to declare exchange %s: %v", mq.ExchangeTypeFanout, err)
	}

	err = c.mqClient.DeclareExchange(mq.ExchangeAppTopic, mq.ExchangeTypeTopic)
	if err != nil {
		log.Fatalf("❌ Failed to declare exchange %s: %v", mq.ExchangeAppTopic, err)
	}

	err = c.mqClient.DeclareExchange(mq.ExchangeMatchEvents, mq.ExchangeTypeFanout)
	if err != nil {
		log.Fatalf("❌ Failed to declare exchange %s: %v", mq.ExchangeMatchEvents, err)
	}

	// Queue 생성 및 바인딩
	queue, err := c.mqClient.DeclareQueue(mq.QueueChat, mq.ExchangeAppTopic,
		[]string{
			mq.RoutingKeyChat,
			mq.RoutingKeyRoomTimeout,
			mq.RoutingKeyFinalChoiceTimeout,
			mq.RoutingKeyRoomJoin,
		})
	if err != nil {
		log.Fatalf("❌ Failed to declare queue %s for %s: %v", mq.QueueChat, mq.ExchangeAppTopic, err)
	}

	_, err = c.mqClient.DeclareQueue(mq.QueueChat, mq.ExchangeMatchEvents, []string{})
	if err != nil {
		log.Fatalf("❌ Failed to declare queue %s for %s: %v", mq.QueueChat, mq.ExchangeMatchEvents, err)
	}

	_, err = c.mqClient.DeclareQueue(mq.QueueChat, mq.ExchangeChatRoomCreateEvents, []string{})
	if err != nil {
		log.Fatalf("❌ Failed to declare queue %s for %s: %v", mq.QueueChat, mq.ExchangeChatRoomCreateEvents, err)
	}

	// 이벤트 핸들러 등록
	handlers := mq.EventHandlerMap{
		mq.EventTypeChat:               c.eventHandler.HandleChatEvent,
		mq.EventTypeMatch:              c.eventHandler.HandleMatchEvent,
		mq.EventTypeRoomCreate:         c.eventHandler.HandleRoomCreateEvent,
		mq.EventTypeRoomTimeout:        c.eventHandler.HandleRoomTimeout,
		mq.EventTypeFinalChoiceTimeout: c.eventHandler.HandleFinalChoiceTimeout,
		mq.EventTypeRoomJoin:           c.eventHandler.HandleRoomJoin,
	}

	// 메시지 소비 시작
	c.mqClient.ConsumeMessages(queue.Name, handlers)

	log.Println("✅ RabbitMQ Consumer Listening...")
}
