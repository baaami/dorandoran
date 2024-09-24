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
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// 유저 서비스 관련 라우팅
	mux.Get("/read/{id}", app.readUser)
	mux.Get("/exist", app.checkUserExistence)
	mux.Post("/register", app.registerUser)
	mux.Put("/update/{id}", app.updateUser)
	mux.Delete("/delete/{id}", app.deleteUser)

	return mux
}
