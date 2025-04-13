package event

import (
	"log"
	"solo/pkg/mq"
	"solo/pkg/redis"
	eventtypes "solo/pkg/types/eventtype"
	"solo/services/game/service"
)

type Consumer struct {
	mqClient     *mq.RabbitMQ
	eventHandler *EventHandler
}

func NewConsumer(mqClient *mq.RabbitMQ, redisClient *redis.RedisClient, gameService *service.GameService) *Consumer {
	return &Consumer{
		mqClient:     mqClient,
		eventHandler: NewEventHandler(gameService, redisClient),
	}
}

func (c *Consumer) StartListening() {
	// Exchange 및 Queue 설정
	err := c.mqClient.DeclareExchange(mq.ExchangeAppTopic, mq.ExchangeTypeTopic)
	if err != nil {
		log.Fatalf("❌ Failed to declare exchange %s: %v", mq.ExchangeAppTopic, err)
	}

	err = c.mqClient.DeclareExchange(mq.ExchangeCoupleRoomCreateEvents, mq.ExchangeTypeFanout)
	if err != nil {
		log.Fatalf("❌ Failed to declare exchange %s: %v", mq.ExchangeCoupleRoomCreateEvents, err)
	}

	// Queue 생성 및 바인딩
	queue, err := c.mqClient.DeclareQueue(mq.QueueGame, mq.ExchangeAppTopic,
		[]string{
			mq.RoutingKeyChat,
			mq.RoutingKeyChatLatest,
			mq.RoutingKeyRoomLeave,
			mq.RoutingKeyRoomTimeout,
			mq.RoutingKeyVoteCommentChat,
			mq.RoutingKeyFinalChoiceTimeout,
		})
	if err != nil {
		log.Fatalf("❌ Failed to declare queue %s for %s: %v", mq.QueueGame, mq.ExchangeAppTopic, err)
	}

	_, err = c.mqClient.DeclareQueue(mq.QueueGame, mq.ExchangeCoupleRoomCreateEvents, []string{})
	if err != nil {
		log.Fatalf("❌ Failed to declare queue %s for %s: %v", mq.QueueGame, mq.ExchangeCoupleRoomCreateEvents, err)
	}

	// 이벤트 핸들러 등록
	handlers := mq.EventHandlerMap{
		eventtypes.EventTypeChat:               c.eventHandler.HandleChatEvent,
		eventtypes.EventTypeChatLatest:         c.eventHandler.HandleChatLatestEvent,
		eventtypes.EventTypeCoupleRoomCreate:   c.eventHandler.HandleCoupleRoomCreateEvent,
		eventtypes.EventTypeRoomLeave:          c.eventHandler.HandleRoomLeaveEvent,
		eventtypes.EventTypeRoomTimeout:        c.eventHandler.HandleRoomTimeoutEvent,
		eventtypes.EventTypeFinalChoiceTimeout: c.eventHandler.HandleFinalChoiceTimeoutEvent,
		eventtypes.EventTypeVoteCommentChat:    c.eventHandler.HandleVoteCommentChatEvent,
	}

	// 메시지 소비 시작
	c.mqClient.ConsumeMessages(queue.Name, handlers)

	log.Println("✅ RabbitMQ Consumer Listening...")
}
