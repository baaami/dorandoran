package middleware

import (
	"net/http"
	"solo/services/chat/service"
	"strconv"

	"github.com/labstack/echo/v4"
)

func RoomAccessChecker(chatService *service.ChatService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// X-User-ID 헤더에서 유저 ID 가져오기
			userIDStr := c.Request().Header.Get("X-User-ID")
			if userIDStr == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "User ID is required")
			}

			userID, err := strconv.Atoi(userIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid User ID")
			}

			// URL 파라미터에서 방 ID 가져오기
			roomID := c.Param("id")

			// 사용자가 해당 방에 속해있는지 확인
			hasAccess, err := chatService.IsUserInRoom(userID, roomID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check room access")
			}

			if !hasAccess {
				return echo.NewHTTPError(http.StatusForbidden, "You don't have access to this room")
			}

			return next(c)
		}
	}
}
