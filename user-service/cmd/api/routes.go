package main

import (
	"log"
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

	mux.Use(LogRequestURL)

	// 유저 서비스 관련 라우팅
	mux.Get("/find/{id}", app.findUser)
	mux.Get("/exist", app.checkUserExistence)
	mux.Post("/register", app.registerUser)
	mux.Put("/update", app.updateUser)
	mux.Delete("/delete", app.deleteUser)

	return mux
}

// LogRequestURL 미들웨어는 요청의 URL 경로를 출력하는 미들웨어입니다.
func LogRequestURL(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// URL 경로 출력
		log.Printf("User Router: %s %s", r.Method, r.URL.Path)

		// 다음 핸들러로 요청 전달
		next.ServeHTTP(w, r)
	})
}
