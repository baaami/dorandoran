package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	_middleware "github.com/baaami/dorandoran/match-socket-service/pkg/middleware"
	"github.com/baaami/dorandoran/match-socket-service/pkg/redis"
)

func (app *Config) routes(redisClient *redis.RedisClient) http.Handler {
	e := echo.New()

	// CORS 설정
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"https://*", "http://*"},
		AllowMethods:     []string{echo.GET, echo.POST, echo.PUT, echo.PATCH, echo.DELETE, echo.OPTIONS},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	e.Use(_middleware.SessionMiddleware(redisClient))

	// Define WebSocket route for matching
	e.GET("/ws/match", app.HandleMatchSocket)

	return e
}
