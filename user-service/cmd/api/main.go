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
	"github.com/baaami/dorandoran/user/pkg/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

const webPort = 80

type Config struct {
	Models *data.UserService
	Rabbit *amqp.Connection
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

	// Config 구조체 초기화
	app := Config{
		Models: &data.UserService{DB: mysqlClient},
		Rabbit: rabbitConn,
	}

	// DB 초기화 (데이터베이스 및 테이블 생성)
	err = app.Models.InitDB()
	if err != nil {
		log.Panic(err)
	}

	// HTTP 서버 설정
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: app.routes(),
	}

	eventChannel := make(chan types.MatchEvent)

	consumerExchanges := []event.ExchangeConfig{
		{Name: "match_events", Type: "fanout"},
	}

	consumer, err := event.NewConsumer(rabbitConn, consumerExchanges)
	if err != nil {
		log.Printf("Failed to make new match consumer: %v", err)
		os.Exit(1)
	}

	// 핸들러 설정
	handlers := map[string]event.MessageHandler{
		"match": event.MatchEventHandler,
	}

	go func() {
		log.Println("Starting RabbitMQ consumer for matching events")
		if err := consumer.Listen(handlers, eventChannel); err != nil {
			log.Printf("Failed to start RabbitMQ consumer: %v", err)
			os.Exit(1)
		}
	}()

	go func() {
		log.Println("Starting Decrease Game Point Routine By Game Start")
		app.decreaseGamePointByGameStart(eventChannel)
	}()

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
