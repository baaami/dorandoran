package event

import (
	"encoding/json"
	"log"

	"github.com/baaami/dorandoran/chat-socket-service/pkg/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	conn           *amqp.Connection
	routingConfigs []RoutingConfig // Exchange와 Routing Key 설정
}

// NewConsumer 함수: RabbitMQ Consumer 초기화
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
		return err
	}
	defer channel.Close()

	// 모든 Exchanges 선언
	for _, config := range c.routingConfigs {
		err := DeclareExchange(channel, config.Exchange)
		if err != nil {
			log.Printf("Failed to declare exchange %s: %v", config.Exchange.Name, err)
			return err
		}
	}
	return nil
}

// Listen 함수
func (c *Consumer) Listen(handlers map[string]MessageHandler, eventChannel chan<- types.WebSocketMessage) error {
	channel, err := c.conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	for _, config := range c.routingConfigs {
		queue, err := channel.QueueDeclare("chat_socket_queue", false, false, true, false, nil)
		if err != nil {
			return err
		}

		// topic exchnage
		if config.Exchange.Type == "topic" {
			for _, key := range config.Keys {
				err := channel.QueueBind(queue.Name, key, config.Exchange.Name, false, nil)
				if err != nil {
					log.Printf("Failed to bind queue %s to routing key %s: %v", queue.Name, key, err)
					return err
				}
			}
		} else if config.Exchange.Type == "fanout" {
			// fanout exchnage
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
				var payload types.EventPayload
				if err := json.Unmarshal(d.Body, &payload); err != nil {
					log.Printf("Failed to unmarshal message: %v", err)
					continue
				}

				if handler, ok := handlers[payload.EventType]; ok {
					handler(payload, eventChannel)
				} else {
					log.Printf("No handler for event type: %s", payload.EventType)
				}
			}
		}()
	}
	select {} // Block forever
}

type MessageHandler func(payload types.EventPayload, eventChannel chan<- types.WebSocketMessage)

// event: chat
func ChatMessageHandler(payload types.EventPayload, eventChannel chan<- types.WebSocketMessage) {
	var chatMsg types.ChatEvent
	if err := json.Unmarshal(payload.Data, &chatMsg); err != nil {
		log.Printf("Failed to unmarshal chat event: %v", err)
		return
	}

	wsMessage := types.WebSocketMessage{
		Kind:    types.MessageKindMessage,
		Payload: json.RawMessage(payload.Data),
	}
	eventChannel <- wsMessage
}

// event: chat.latest
func ChatLatestHandler(payload types.EventPayload, eventChannel chan<- types.WebSocketMessage) {
	var chatLatest types.ChatLatestEvent
	if err := json.Unmarshal(payload.Data, &chatLatest); err != nil {
		log.Printf("Failed to unmarshal chat.latest event: %v", err)
		return
	}

	wsMessage := types.WebSocketMessage{
		Kind:    types.MessageKindChatLastest,
		Payload: json.RawMessage(payload.Data),
	}
	eventChannel <- wsMessage
}

// event: room.leave
func RoomLeaveHandler(payload types.EventPayload, eventChannel chan<- types.WebSocketMessage) {
	var roomLeave RoomLeaveEvent
	if err := json.Unmarshal(payload.Data, &roomLeave); err != nil {
		log.Printf("Failed to unmarshal room.leave event: %v", err)
		return
	}

	wsMessage := types.WebSocketMessage{
		Kind:    types.MessageKindLeave,
		Payload: json.RawMessage(payload.Data),
	}
	eventChannel <- wsMessage
}

// event: room.couple.create
func CreateCoupleRoomHandler(payload types.EventPayload, eventChannel chan<- types.WebSocketMessage) {
	var coupleRoomEvent types.ChatRoom
	if err := json.Unmarshal(payload.Data, &coupleRoomEvent); err != nil {
		log.Printf("Failed to unmarshal chat.latest event: %v", err)
		return
	}

	wsMessage := types.WebSocketMessage{
		Kind:    types.MessageKindCoupleMatchSuccess,
		Payload: json.RawMessage(payload.Data),
	}
	eventChannel <- wsMessage
}
