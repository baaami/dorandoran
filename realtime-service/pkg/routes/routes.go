package routes

import (
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	socketio "github.com/googollee/go-socket.io"
)

// InitRoutes initializes the HTTP routes and applies CORS settings.
func InitRoutes(socketServer *socketio.Server) http.Handler {
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

	// 요청 경로를 출력하는 미들웨어 추가
	mux.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Incoming request path: %s", r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	// "/socket.io/" 경로에 Socket.IO 서버 연결
	mux.Handle("/socket.io/*", socketServer)

	return mux
}
