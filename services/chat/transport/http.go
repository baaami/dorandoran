package transport

import (
	"solo/pkg/middleware"
	"solo/services/chat/handler"
	"solo/services/chat/service"

	"github.com/labstack/echo/v4"
)

func NewRouter(chatHandler *handler.ChatHandler, chatService *service.ChatService) *echo.Echo {
	e := echo.New()

	// 채팅방 목록 조회는 미들웨어 없이 접근 가능
	e.GET("/room/list", chatHandler.GetChatRoomList)

	// roomID를 사용하는 엔드포인트들을 그룹으로 묶어서 미들웨어 적용
	roomGroup := e.Group("")
	roomGroup.Use(middleware.RoomAccessChecker(chatService))
	{
		// 채팅방 관련 라우팅 (roomID 파라미터 사용)
		roomGroup.GET("/room/:id", chatHandler.GetChatRoomByID)
		roomGroup.DELETE("/room/delete/:id", chatHandler.DeleteChatRoom)
		roomGroup.GET("/list/:id", chatHandler.GetChatMsgListByRoomID)
		roomGroup.DELETE("/all/:id", chatHandler.DeleteChatByRoomID)
		roomGroup.GET("/character/name/:id", chatHandler.GetCharacterNameByRoomID)
	}

	return e
}
