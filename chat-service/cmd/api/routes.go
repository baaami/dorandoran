package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func (app *Config) routes() http.Handler {
	mux := chi.NewRouter()

	// CORS 설정
	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// 채팅방 관련 라우팅
	mux.Post("/room/create", app.createChatRoom)
	mux.Get("/room/list", app.getChatRooms)
	mux.Delete("/room/delete/{id}", app.deleteChatRoom)

	// 채팅 메시지
	mux.Post("/msg", app.addChatMsg)
	mux.Get("/list/{id}", app.getChatMsgList)

	return mux
}
