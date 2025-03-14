package main

import (
	"fmt"
	"log"
	"net/http"
	"solo/pkg/redis"
	"solo/services/gateway/handler"
	"solo/services/gateway/transport"
)

const webPort = 80

func main() {
	redisClient, err := redis.NewRedisClient()
	if err != nil {
		log.Panic("Redis 연결 실패: ", err)
	}

	gatewayHandler := handler.NewGatewayHandler()

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: transport.NewRouter(gatewayHandler, redisClient),
	}

	log.Printf("🚀 Gateway Service Started on Port %d", webPort)
	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
