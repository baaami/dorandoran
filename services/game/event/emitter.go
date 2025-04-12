package event

import (
	"encoding/json"
	"log"
	"solo/pkg/helper"
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

func (e *Emitter) PublishRoomJoinEvent(event eventtypes.RoomJoinEvent) error {
	payload := eventtypes.EventPayload{
		EventType: eventtypes.EventTypeRoomJoin,
		Data:      helper.ToJSON(event),
	}
	return e.publish(mq.ExchangeAppTopic, mq.RoutingKeyRoomJoin, payload)
}

func (e *Emitter) PublishChatMessageEvent(event eventtypes.ChatEvent) error {
	payload := eventtypes.EventPayload{
		EventType: eventtypes.EventTypeChat,
		Data:      helper.ToJSON(event),
	}

	err := e.publish(mq.ExchangeAppTopic, mq.RoutingKeyChat, payload)
	if err != nil {
		log.Printf("❌ Failed to publish ChatMessageEvent: %v", err)
		return err
	}

	log.Printf("📢 Published ChatMessageEvent for RoomID: %s", event.RoomID)
	return nil
}

func (e *Emitter) PublishFinalChoiceTimeoutEvent(event eventtypes.FinalChoiceTimeoutEvent) error {
	payload := eventtypes.EventPayload{
		EventType: eventtypes.EventTypeFinalChoiceTimeout,
		Data:      helper.ToJSON(event),
	}

	err := e.publish(mq.ExchangeAppTopic, mq.RoutingKeyFinalChoiceTimeout, payload)
	if err != nil {
		log.Printf("❌ Failed to publish final choice timeout event: %v", err)
		return err
	}

	log.Printf("📢 Published final choice timeout event to RoomID: %s", event.RoomID)
	return nil
}

func (e *Emitter) PublishRoomTimeoutEvent(timeoutEvent eventtypes.RoomTimeoutEvent) error {
	payload := eventtypes.EventPayload{
		EventType: eventtypes.EventTypeRoomTimeout,
		Data:      helper.ToJSON(timeoutEvent),
	}
	return e.publish(mq.ExchangeAppTopic, mq.RoutingKeyRoomTimeout, payload)
}

func (e *Emitter) PublishMatchEvent(event eventtypes.MatchEvent) error {
	payload := eventtypes.EventPayload{
		EventType: eventtypes.EventTypeMatch,
		Data:      helper.ToJSON(event),
	}
	return e.publish(mq.ExchangeMatchEvents, "", payload)
}
