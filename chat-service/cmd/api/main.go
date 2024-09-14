package main

import (
	"fmt"
	"log"
	"net/http"
)

const webPort = 80

type Config struct{}

func main() {
	// Config 구조체 생성
	app := Config{}

	// HTTP 서버 설정
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: app.routes(),
	}

	// 서버 시작
	log.Printf("Starting Chat Service on port %d", webPort)
	err := srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
