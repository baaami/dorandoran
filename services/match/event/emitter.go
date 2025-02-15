package event

import (
	"encoding/json"
	"log"
	"solo/pkg/mq"
	eventtypes "solo/pkg/types/eventtype"
)

type Emitter struct {
	mqClient *mq.RabbitMQ
}

func NewEmitter(mqClient *mq.RabbitMQ) *Emitter {
	return &Emitter{mqClient: mqClient}
}

func (e *Emitter) PublishMatchEvent(payload eventtypes.EventPayload) error {
	eventBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("❌ Failed to marshal match success event: %v", err)
		return err
	}

	err = e.mqClient.PublishMessage(
		mq.ExchangeMatchEvents, // Exchange Name (Fanout 타입)
		"",                     // Routing Key (Fanout은 필요 없음)
		eventBytes,
	)
	if err != nil {
		log.Printf("❌ Failed to publish match success event: %v", err)
		return err
	}

	log.Printf("Match success event published")
	return nil
}
