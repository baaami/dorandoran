package transport

import (
	"solo/services/auth/handler"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// RegisterAuthRoutes 설정
func RegisterAuthRoutes(e *echo.Echo, authHandler *handler.AuthHandler) {
	// CORS 설정
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"https://*", "http://*"},
		AllowMethods:     []string{echo.GET, echo.POST, echo.PUT, echo.PATCH, echo.DELETE, echo.OPTIONS},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// 로그인 API
	e.POST("/kakao", authHandler.KakaoLoginHandler)
	e.POST("/naver", authHandler.NaverLoginHandler)
}
