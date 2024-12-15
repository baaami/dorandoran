package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/baaami/dorandoran/match-socket-service/pkg/redis"
)

const webPort = 80

type Config struct {
	MatchClients sync.Map // key: userID, value: *websocket.Conn
	RedisClient  *redis.RedisClient
}

func main() {
	// Redis 연결
	redisClient, err := redis.NewRedisClient()
	if err != nil {
		log.Fatalf("Failed to initialize Redis client: %v", err)
	}

	// Config 구조체 생성
	app := Config{
		RedisClient: redisClient,
	}

	// HTTP 서버 설정
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: app.routes(),
	}

	// 서버 시작
	log.Printf("Starting Match Socket Service on port %d", webPort)
	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
