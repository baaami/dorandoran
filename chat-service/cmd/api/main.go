package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/baaami/dorandoran/chat/cmd/data"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	webPort  = "80"
	mongoURL = "mongodb://mongo:27017"
)

var client *mongo.Client

type Config struct {
	Models data.Models
}

func main() {
	// MongoDB 연결
	mongoClient, err := connectToMongo()
	if err != nil {
		log.Panic(err)
	}
	client = mongoClient

	// MongoDB 연결 해제 시 사용되는 컨텍스트 생성
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// MongoDB 연결 해제
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	// Config 구조체 초기화
	app := Config{
		Models: data.New(client),
	}

	// 웹 서버 시작
	log.Println("Starting Chat Service on port", webPort)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.routes(),
	}

	err = srv.ListenAndServe()
	if err != nil {
		log.Panic()
	}
}

// MongoDB에 연결하는 함수
func connectToMongo() (*mongo.Client, error) {
	// MongoDB 연결 옵션 설정
	clientOptions := options.Client().ApplyURI(mongoURL)
	clientOptions.SetAuth(options.Credential{
		Username: "admin",
		Password: "sample",
	})

	// MongoDB에 연결
	c, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Println("Error connecting to MongoDB:", err)
		return nil, err
	}

	log.Println("Connected to MongoDB!")

	return c, nil
}
