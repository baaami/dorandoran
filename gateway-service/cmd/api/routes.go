package main

import (
	"log"
	"net/http"

	"github.com/baaami/dorandoran/broker/pkg/middleware"
	"github.com/baaami/dorandoran/broker/pkg/redis"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func (app *Config) routes(redisClient *redis.RedisClient) http.Handler {
	mux := chi.NewRouter()

	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	mux.Use(LogRequestURL)
	// 세션 검증 미들웨어 추가
	mux.Use(middleware.SessionMiddleware(redisClient))

	// WebSocket 프록시 라우팅
	mux.Handle("/ws/match", app.proxySocketServer("ws://match-socket-service"))
	mux.Handle("/ws/chat", app.proxySocketServer("ws://chat-socket-service"))

	// 프로필 이미지
	mux.Get("/profile", app.profileHandler)

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

// LogRequestURL 미들웨어는 요청의 URL 경로를 출력하는 미들웨어입니다.
func LogRequestURL(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// URL 경로 출력
		log.Printf("API Gateway: %s %s", r.Method, r.URL.Path)

		// 다음 핸들러로 요청 전달
		next.ServeHTTP(w, r)
	})
}
