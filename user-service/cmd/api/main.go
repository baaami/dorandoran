package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/baaami/dorandoran/user/cmd/data"
	"github.com/baaami/dorandoran/user/pkg/event"
	amqp "github.com/rabbitmq/amqp091-go"
)

const webPort = 80

type Config struct {
	Models       *data.UserService
	Rabbit       *amqp.Connection
	EventChannel chan event.EventPayload
}

func main() {
	mysqlClient, err := connectToMySQL()
	if err != nil {
		log.Panic(err)
	}

	rabbitConn, err := connectToRabbitMQ()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer rabbitConn.Close()

	eventChannel := make(chan event.EventPayload, 10)

	// Config 구조체 초기화
	app := Config{
		Models:       &data.UserService{DB: mysqlClient},
		Rabbit:       rabbitConn,
		EventChannel: eventChannel,
	}

	// DB 초기화 (데이터베이스 및 테이블 생성)
	err = app.Models.InitDB()
	if err != nil {
		log.Panic(err)
	}

	routingConfigs := []event.RoutingConfig{
		{
			Exchange: event.ExchangeConfig{Name: event.ExchangeAppTopic, Type: "topic"},
			Keys:     []string{event.EventTypeFinalChoiceTimeout},
		},
		{
			Exchange: event.ExchangeConfig{Name: event.ExchangeMatchEvents, Type: "fanout"},
			Keys:     []string{}, // fanout 타입은 라우팅 키가 필요 없음
		},
	}

	consumer, err := event.NewConsumer(rabbitConn, routingConfigs)
	if err != nil {
		log.Printf("Failed to make new match consumer: %v", err)
		os.Exit(1)
	}

	go func() {
		log.Println("Starting RabbitMQ consumer for matching events")
		if err := consumer.Listen(eventChannel); err != nil {
			log.Printf("Failed to start RabbitMQ consumer: %v", err)
			os.Exit(1)
		}
	}()

	// event -> handler
	go app.EventPayloadHandler()

	// HTTP 서버 설정
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: app.routes(),
	}

	// 서버 시작
	log.Printf("Starting User Service on port %d", webPort)
	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

// MySQL에 연결하는 함수
func connectToMySQL() (*gorm.DB, error) {
	dsn := "root:sample@tcp(mysql:3306)/users?parseTime=true"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 로깅 설정
	})
	if err != nil {
		log.Println("Error connecting to MySQL:", err)
		return nil, err
	}

	log.Println("Connected to MySQL!")

	return db, nil
}

func connectToRabbitMQ() (*amqp.Connection, error) {
	return amqp.Dial("amqp://guest:guest@rabbitmq")
}
