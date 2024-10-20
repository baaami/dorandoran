package event

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	common "github.com/baaami/dorandoran/common/chat"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RoomJoinEvent struct {
	RoomID string `bson:"room_id" json:"room_id"`
	UserID string `bson:"user_id" json:"user_id"`
}

// Chat 구조체 정의
type Chat struct {
	RoomID    string    `bson:"room_id" json:"room_id"`
	SenderID  string    `bson:"sender_id" json:"sender_id"`
	Message   string    `bson:"message" json:"message"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

// User 구조체 정의
type Address struct {
	City     string `gorm:"size:100" json:"city"`
	District string `gorm:"size:100" json:"district"`
	Street   string `gorm:"size:100" json:"street"`
}

type User struct {
	ID      int     `gorm:"primaryKey;autoIncrement" json:"id"`
	SnsType int     `gorm:"index" json:"sns_type"`
	SnsID   int64   `gorm:"index" json:"sns_id"`
	Name    string  `gorm:"size:100" json:"name"`
	Gender  int     `json:"gender"`
	Birth   string  `gorm:"size:20" json:"birth"`
	Address Address `gorm:"embedded;embeddedPrefix:address_" json:"address"`
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
	return declareExchange(channel)
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
			"app_topic", // 이벤트를 수신할 exchange
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
				var chatMsg Chat
				// eventPayload.Data는 json.RawMessage이므로 다시 언마샬링
				if err := json.Unmarshal(eventPayload.Data, &chatMsg); err != nil {
					log.Printf("Failed to unmarshal chat message: %v", err)
					continue
				}
				log.Printf("Chat Message Unmarshaled: %+v", chatMsg)
				handleChatAddPayload(chatMsg)

			case "user.created":
				var user User
				if err := json.Unmarshal(eventPayload.Data, &user); err != nil {
					log.Printf("Failed to unmarshal user message: %v", err)
					continue
				}
				log.Printf("User Created Message Unmarshaled: %+v", user)
				handleUserCreatedEvent(user)

			case "room.join":
				var roomJoin RoomJoinEvent
				if err := json.Unmarshal(eventPayload.Data, &roomJoin); err != nil {
					log.Printf("Failed to unmarshal room join event: %v", err)
					continue
				}
				// TODO: room join event 발생 시점에 아직 room이 동기적으로 생성되지 않음
				// handleRoomJoinEvent(roomJoin)

			case "room.deleted":
				var room common.ChatRoom
				if err := json.Unmarshal(eventPayload.Data, &room); err != nil {
					log.Printf("Failed to unmarshal room message: %v", err)
					continue
				}
				handleRoomDeletedEvent(room)
			default:
				log.Printf("Unknown event type: %s", eventPayload.EventType)
			}
		}
	}()

	fmt.Printf("Waiting for messages [Exchange: app_topic, Queue: %s]\n", q.Name)
	<-forever

	return nil
}

// handleChatPayload는 채팅 메시지를 처리하는 함수
func handleChatAddPayload(chatMsg Chat) error {
	jsonData, _ := json.MarshalIndent(&chatMsg, "", "\t")

	chatServiceURL := "http://chat-service/msg"

	request, err := http.NewRequest(http.MethodPost, chatServiceURL, bytes.NewBuffer(jsonData))
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
func handleUserCreatedEvent(user User) error {
	jsonData, _ := json.MarshalIndent(&user, "", "\t")

	userServiceURL := "http://user-service/user/insert"

	request, err := http.NewRequest(http.MethodPost, userServiceURL, bytes.NewBuffer(jsonData))
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

// 채팅방 참가 시 이벤트 발생 동작
func handleRoomJoinEvent(roomJoin RoomJoinEvent) error {
	// 채팅방에서 유저가 마지막으로 확인한 시간 업데이트
	url := fmt.Sprintf("http://chat-service/room/confirm/%s/%s", roomJoin.RoomID, roomJoin.UserID)

	request, err := http.NewRequest(http.MethodPut, url, nil)
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

	if response.StatusCode != http.StatusOK {
		log.Printf("Failed to send room confirm, status: %d, err: %v", response.StatusCode, err)
		return err
	}

	log.Println("Room join event successfully consumed")
	return nil
}

// 채팅방 삭제 이벤트 발생 시 동작
func handleRoomDeletedEvent(room common.ChatRoom) error {
	// 채팅방에서 사용한 채팅 데이터 삭제
	url := fmt.Sprintf("http://chat-service/all/%s", room.ID)

	request, err := http.NewRequest(http.MethodDelete, url, nil)
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

	if response.StatusCode != http.StatusOK {
		log.Printf("Failed to send chat delete all, status: %d, err: %v", response.StatusCode, err)
		return err
	}

	log.Println("Room delete event successfully consumed")
	return nil
}
