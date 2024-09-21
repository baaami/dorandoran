package main

import (
	"net/http"

	"github.com/baaami/dorandoran/broker/pkg/socket"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func (app *Config) routes(wsConfig *socket.Config) http.Handler {
	mux := chi.NewRouter()

	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	mux.HandleFunc("/ws", wsConfig.HandleWebSocket)

	// 매칭
	mux.Handle("/match/*", http.HandlerFunc(app.proxyService()))

	// 로그인
	mux.Handle("/auth/*", http.HandlerFunc(app.proxyService()))

	// 유저 정보
	mux.Handle("/user/*", http.HandlerFunc(app.proxyService()))

	// 채팅
	mux.Handle("/chat/*", http.HandlerFunc(app.proxyService()))

	// API 명세
	mux.Get("/", app.usage)

	return mux
}
