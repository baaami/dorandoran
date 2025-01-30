package event

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/baaami/dorandoran/push/pkg/onesignal"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	conn           *amqp.Connection
	routingConfigs []RoutingConfig
}

func NewConsumer(conn *amqp.Connection, routingConfigs []RoutingConfig) (*Consumer, error) {
	consumer := &Consumer{
		conn:           conn,
		routingConfigs: routingConfigs,
	}

	err := consumer.setup()
	if err != nil {
		log.Printf("Failed to setup consumer: %v", err)
		return nil, err
	}

	return consumer, nil
}

func (c *Consumer) setup() error {
	channel, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %v", err)
	}
	defer channel.Close()

	for _, config := range c.routingConfigs {
		err := DeclareExchange(channel, config.Exchange)
		if err != nil {
			log.Printf("Failed to declare exchange %s: %v", config.Exchange.Name, err)
			return err
		}
	}
	return nil
}

func (c *Consumer) Listen() error {
	channel, err := c.conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	for _, config := range c.routingConfigs {
		queue, err := channel.QueueDeclare("push_queue", false, false, true, false, nil)
		if err != nil {
			return err
		}

		if config.Exchange.Type == "topic" {
			for _, key := range config.Keys {
				err := channel.QueueBind(queue.Name, key, config.Exchange.Name, false, nil)
				if err != nil {
					log.Printf("Failed to bind queue %s to routing key %s: %v", queue.Name, key, err)
					return err
				}
			}
		} else if config.Exchange.Type == "fanout" {
			err := channel.QueueBind(queue.Name, "", config.Exchange.Name, false, nil)
			if err != nil {
				log.Printf("Failed to bind queue %s to routing key %s: %v", queue.Name, "", err)
				return err
			}
		}

		messages, err := channel.Consume(queue.Name, "", true, false, false, false, nil)
		if err != nil {
			return err
		}

		go func() {
			for d := range messages {
				var eventPayload EventPayload
				if err := json.Unmarshal(d.Body, &eventPayload); err != nil {
					log.Printf("Failed to unmarshal message: %v", err)
					continue
				}

				log.Printf("Received Event Type: %s", eventPayload.EventType)

				// handling
				switch eventPayload.EventType {
				case EventTypeChat:
					handleChatEvent(eventPayload.Data)
				case EventTypeRoomTimeout:
					handleRoomTimeoutEvent(eventPayload.Data)
				}
			}
		}()
	}
	select {} // Block forever
}

func handleChatEvent(jsonData json.RawMessage) error {
	var chatMsg ChatEvent
	if err := json.Unmarshal(jsonData, &chatMsg); err != nil {
		return fmt.Errorf("failed to unmarshal chat data: %w", err)
	}

	payload := onesignal.Payload{
		PushUserList: chatMsg.InactiveUserIds,
		Header:       "New Message",
		Content:      chatMsg.Message,
		Url:          fmt.Sprintf("randomChat://game-room/%s", chatMsg.RoomID),
	}

	onesignal.Push(payload)

	return nil
}

func handleRoomTimeoutEvent(jsonData json.RawMessage) error {
	var roomTimeoutEvent RoomTimeoutEvent
	err := json.Unmarshal(jsonData, &roomTimeoutEvent)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoomTimeoutEvent: %v", err)
	}

	payload := onesignal.Payload{
		PushUserList: roomTimeoutEvent.InactiveUserIds,
		Header:       "Final Choice Start",
		Content:      "Final Choice Start",
		Url:          fmt.Sprintf("randomChat://game-room/%s", roomTimeoutEvent.RoomID),
	}

	onesignal.Push(payload)

	return nil
}
