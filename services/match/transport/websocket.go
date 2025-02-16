package transport

import (
	"solo/pkg/middleware"
	"solo/pkg/redis"
	"solo/services/match/handler"

	"github.com/labstack/echo/v4"
	echo_middleware "github.com/labstack/echo/v4/middleware"
)

func NewRouter(matchHandler *handler.MatchHandler, redisClient *redis.RedisClient) *echo.Echo {
	e := echo.New()

	e.Use(echo_middleware.CORSWithConfig(echo_middleware.CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{echo.GET, echo.POST, echo.PUT, echo.PATCH, echo.DELETE, echo.OPTIONS},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	e.Use(middleware.SessionMiddleware(redisClient))

	e.GET("/ws/match", matchHandler.HandleMatchSocket)

	return e
}
