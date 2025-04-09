package event

import (
	"encoding/json"
	"log"
	"solo/pkg/helper"
	"solo/pkg/models"
	"solo/pkg/mq"
	eventtypes "solo/pkg/types/eventtype"
)

type Emitter struct {
	mqClient *mq.RabbitMQ
}

func NewEmitter(mqClient *mq.RabbitMQ) *Emitter {
	return &Emitter{mqClient: mqClient}
}

func (e *Emitter) publish(exchangeName, routingKey string, payload eventtypes.EventPayload) error {
	eventBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("❌ Failed to marshal event: %s, err: %v", payload.EventType, err)
		return err
	}

	err = e.mqClient.PublishMessage(
		exchangeName, // Exchange Name (Fanout 타입)
		routingKey,   // Routing Key (Fanout은 필요 없음)
		eventBytes,
	)
	if err != nil {
		log.Printf("❌ Failed to publish event: %s, err: %v", payload.EventType, err)
		return err
	}

	return nil
}

func (e *Emitter) PublishChatRoomCreateEvent(data models.ChatRoom) error {
	payload := eventtypes.EventPayload{
		EventType: eventtypes.EventTypeRoomCreate,
		Data:      helper.ToJSON(data),
	}
	return e.publish(mq.ExchangeChatRoomCreateEvents, "", payload)
}

func (e *Emitter) PublishCoupleRoomCreateEvent(data models.ChatRoom) error {
	payload := eventtypes.EventPayload{
		EventType: eventtypes.EventTypeCoupleRoomCreate,
		Data:      helper.ToJSON(data),
	}
	return e.publish(mq.ExchangeCoupleRoomCreateEvents, "", payload)
}

func (e *Emitter) PublishChatLatestEvent(data eventtypes.ChatLatestEvent) error {
	payload := eventtypes.EventPayload{
		EventType: eventtypes.EventTypeChatLatest,
		Data:      helper.ToJSON(data),
	}
	return e.publish(mq.ExchangeAppTopic, mq.RoutingKeyChatLatest, payload)
}

func (e *Emitter) PublishRoomLeaveEvent(data eventtypes.RoomLeaveEvent) error {
	payload := eventtypes.EventPayload{
		EventType: eventtypes.EventTypeRoomLeave,
		Data:      helper.ToJSON(data),
	}
	return e.publish(mq.ExchangeAppTopic, mq.RoutingKeyRoomLeave, payload)
}

func (e *Emitter) PublishVoteCommentChatEvent(event eventtypes.VoteCommentChatEvent) error {
	payload := eventtypes.EventPayload{
		EventType: eventtypes.EventTypeVoteCommentChat,
		Data:      helper.ToJSON(event),
	}
	return e.publish(mq.ExchangeAppTopic, mq.RoutingKeyVoteCommentChat, payload)
}
