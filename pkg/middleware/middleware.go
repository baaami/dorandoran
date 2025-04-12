package middleware

import (
	"net/http"
	"solo/pkg/redis"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

func SessionMiddleware(redisClient *redis.RedisClient) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// TODO: 인증이 필요 없는 경로 처리, 추후 보안 강화 예정
			if strings.Contains(c.Path(), "auth") || strings.Contains(c.Path(), "profile") {
				return next(c)
			}

			// 쿠키에서 세션 ID 추출
			cookie, err := c.Cookie("session_id")
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized: No session ID provided"})
			}
			sessionID := cookie.Value

			// Redis에서 세션 ID로 사용자 정보 조회
			userID, err := redisClient.GetUserBySessionID(sessionID)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized: Invalid session ID"})
			}

			// 사용자 ID를 컨텍스트에 저장
			c.Request().Header.Set("X-User-ID", strconv.Itoa(userID))

			// 다음 핸들러로 요청 전달
			return next(c)
		}
	}
}
