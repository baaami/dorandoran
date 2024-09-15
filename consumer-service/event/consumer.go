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
	SenderID   string `json:"senderID"`
	ReceiverID string `json:"receiverID"`
	ChatRoomID string `json:"chatRoomID"`
	Message    string `json:"message"`
	CreatedAt  string `json:"createdAt"`
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

	return declareChatExchange(channel)
}

type Payload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

// Listen 함수는 RabbitMQ에서 chat 메시지를 수신합니다.
func (consumer *Consumer) Listen(topics []string) error {
	log.Println("Setting up listener for chat topics...")

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
			"chat_topic", // chat_topic exchange로 바인딩
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

			var chatMsg ChatMessage
			err := json.Unmarshal(d.Body, &chatMsg)
			if err != nil {
				log.Printf("Failed to unmarshal message: %v", err)
				continue
			}

			// 채팅 메시지 로그 출력
			log.Printf("Received chat message from %s to %s in room %s: %s", chatMsg.SenderID, chatMsg.ReceiverID, chatMsg.ChatRoomID, chatMsg.Message)
			handleChatPayload(chatMsg)

			// 이후 MongoDB에 저장하는 로직을 여기에 추가 가능
		}
	}()

	fmt.Printf("Waiting for messages [Exchange: chat_topic, Queue: %s]\n", q.Name)
	<-forever

	return nil
}

func handleChatPayload(chatMsg ChatMessage) error {
	payload := ChatMessage{
		SenderID:   chatMsg.SenderID,
		ReceiverID: chatMsg.ReceiverID,
		ChatRoomID: chatMsg.ChatRoomID,
		Message:    chatMsg.Message,
		CreatedAt:  chatMsg.CreatedAt,
	}

	jsonData, _ := json.MarshalIndent(&payload, "", "\t")

	chatServiceURL := "http://chat-service/msg"

	request, err := http.NewRequest("POST", chatServiceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to NewRequest(): %v", err)
		return err
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		log.Printf("Failed to Request: %v", err)
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {
		log.Printf("Failed to Response: %v", err)
		return err
	}

	return nil
}

// TODO: Socket에 맞춰서 payload 형식을 맞춰야함
func handlePayload(payload Payload) {
	switch payload.Name {
	case "log", "event":
		// log whatever we get
		err := logEvent(payload)
		if err != nil {
			log.Println(err)
		}

	case "auth":
		// authenticate
		// you can have as many cases as you want, as long as you write the logic

	default:
		break
	}
}

func logEvent(entry Payload) error {
	jsonData, _ := json.MarshalIndent(entry, "", "\t")

	logServiceURL := "http://logger-service/log"

	request, err := http.NewRequest("POST", logServiceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {
		return err
	}

	return nil
}
