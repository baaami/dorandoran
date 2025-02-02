package transport

import (
	"net/http"

	"solo/services/user/handler"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func NewRouter(userHandler *handler.UserHandler, filterHandler *handler.FilterHandler) http.Handler {
	mux := chi.NewRouter()

	// CORS 설정
	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	mux.Get("/find/list", userHandler.FindUserList)
	mux.Get("/find", userHandler.FindUser)
	mux.Get("/exist", userHandler.CheckUser)

	mux.Post("/register", userHandler.RegisterUser)

	mux.Patch("/update", userHandler.UpdateUser)

	mux.Delete("/delete", userHandler.DeleteUser)

	mux.Get("/match/filter", filterHandler.FindMatchFilter)
	mux.Patch("/match/filter", filterHandler.UpdateMatchFilter)

	return mux
}
