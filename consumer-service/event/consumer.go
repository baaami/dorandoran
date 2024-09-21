package event

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ChatMessage 구조체 정의
type ChatMessage struct {
	RoomID     string `bson:"room_id"`
	SenderID   string `bson:"sender_id"`
	ReceiverID string `bson:"receiver_id"`
	Message    string `bson:"message"`
}

// User 구조체 정의
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Nickname string `json:"nickname"`
	Gender   int    `json:"gender"`
	Age      int    `json:"age"`
	Email    string `json:"email"`
}

type Consumer struct {
	conn      *amqp.Connection
	queueName string
}

func NewConsumer(conn *amqp.Connection) (Consumer, error) {
	consumer := Consumer{
		conn: conn,
	}

	err := consumer.setup()
	if err != nil {
		log.Printf("Failed to setup consumer: %v", err)
		return Consumer{}, err
	}

	return consumer, nil
}

func (consumer *Consumer) setup() error {
	channel, err := consumer.conn.Channel()
	if err != nil {
		return err
	}

	// Exchange 선언
	return declareChatExchange(channel)
}

// EventPayload 구조체 정의
type EventPayload struct {
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

// Listen 함수는 RabbitMQ에서 메시지를 수신하여 이벤트 처리
func (consumer *Consumer) Listen(topics []string) error {
	log.Println("Setting up listener for events...")

	ch, err := consumer.conn.Channel()
	if err != nil {
		log.Printf("Failed to open channel: %v", err)
		return err
	}
	defer ch.Close()

	q, err := declareRandomQueue(ch)
	if err != nil {
		log.Printf("Failed to declare random queue: %v", err)
		return err
	}
	log.Printf("Declared queue: %s", q.Name)

	for _, s := range topics {
		err = ch.QueueBind(
			q.Name,
			s,
			"chat_topic", // 이벤트를 수신할 exchange
			false,
			nil,
		)
		if err != nil {
			log.Printf("Failed to bind queue %s to topic %s: %v", q.Name, s, err)
			return err
		}
		log.Printf("Queue %s bound to topic %s", q.Name, s)
	}

	messages, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Printf("Failed to consume messages: %v", err)
		return err
	}

	log.Println("Listening for messages...")

	forever := make(chan bool)
	go func() {
		for d := range messages {
			log.Printf("Message received: %s", d.Body)

			var eventPayload EventPayload
			err := json.Unmarshal(d.Body, &eventPayload)
			if err != nil {
				log.Printf("Failed to unmarshal event payload: %v", err)
				continue
			}

			log.Printf("Event Type: %s", eventPayload.EventType)

			switch eventPayload.EventType {
			case "chat":
				var chatMsg ChatMessage
				// eventPayload.Data는 json.RawMessage이므로 다시 언마샬링
				if err := json.Unmarshal(eventPayload.Data, &chatMsg); err != nil {
					log.Printf("Failed to unmarshal chat message: %v", err)
					continue
				}
				log.Printf("Chat Message Unmarshaled: %+v", chatMsg)
				handleChatPayload(chatMsg)

			case "user.created":
				var user User
				if err := json.Unmarshal(eventPayload.Data, &user); err != nil {
					log.Printf("Failed to unmarshal user message: %v", err)
					continue
				}
				log.Printf("User Created Message Unmarshaled: %+v", user)
				handleUserPayload(user)

			default:
				log.Printf("Unknown event type: %s", eventPayload.EventType)
			}
		}
	}()

	fmt.Printf("Waiting for messages [Exchange: chat_topic, Queue: %s]\n", q.Name)
	<-forever

	return nil
}

// handleChatPayload는 채팅 메시지를 처리하는 함수
func handleChatPayload(chatMsg ChatMessage) error {
	jsonData, _ := json.MarshalIndent(&chatMsg, "", "\t")

	chatServiceURL := "http://chat-service/msg"

	request, err := http.NewRequest("POST", chatServiceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return err
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		log.Printf("Failed to send request: %v", err)
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		log.Printf("Failed to send chat message: %v", err)
		return err
	}

	return nil
}

// handleUserPayload는 유저 생성 이벤트를 처리하는 함수
func handleUserPayload(user User) error {
	jsonData, _ := json.MarshalIndent(&user, "", "\t")

	userServiceURL := "http://user-service/user/insert"

	request, err := http.NewRequest("POST", userServiceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return err
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		log.Printf("Failed to send request: %v", err)
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		log.Printf("Failed to send user creation event: %v", err)
		return err
	}

	log.Println("User creation event successfully sent to user-service")
	return nil
}
