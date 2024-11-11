package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/baaami/dorandoran/auth/pkg/redis"
)

const webPort = 80

type Config struct {
	RedisClient *redis.RedisClient
}

func main() {
	timeInit()

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
	log.Printf("Starting Auth Service on port %d", webPort)
	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func timeInit() { // KST 설정
	// 서비스 초기화 시 KST를 전역 로케일로 설정
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		panic(fmt.Sprintf("Failed to load KST location: %v", err))
	}
	time.Local = loc
}
