package main

import (
	"fmt"
	"log"
	"net/http"

	socketio "github.com/googollee/go-socket.io"
)

const webPort = 80

type Config struct {
	ws *socketio.Server
}

func main() {
	app := Config{}

	log.Printf("Starting Gateway service on port %d", webPort)

	// 초기화
	app.InitSocket()

	// HTTP 서버 설정
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: app.routes(),
	}

	// 서버 시작
	log.Printf("Starting Gateway Server on port %d", webPort)
	err := srv.ListenAndServe()
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
