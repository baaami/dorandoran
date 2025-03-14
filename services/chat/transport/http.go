package transport

import (
	"solo/services/chat/handler"
	"solo/services/chat/service"

	"github.com/labstack/echo/v4"
)

func NewRouter(chatHandler *handler.ChatHandler, chatService *service.ChatService) *echo.Echo {
	e := echo.New()

	// 채팅방 목록 조회는 미들웨어 없이 접근 가능
	e.GET("/room/list", chatHandler.GetChatRoomList)

	e.GET("/balance/form/:formid", chatHandler.GetBalanceFormByID)
	e.POST("/balance/form/vote/:formid", chatHandler.InsertBalanceFormVote)
	e.DELETE("/balance/form/vote/:formid", chatHandler.CancelBalanceFormVote)
	e.POST("/balance/form/comment/:formid", chatHandler.InsertBalanceFormComment)
	e.GET("/balance/form/comment/:formid", chatHandler.GetBalanceFormComments)

	// 채팅방 관련 라우팅 (roomID 파라미터 사용)
	e.GET("/room/:id", chatHandler.GetChatRoomByID)
	e.DELETE("/room/delete/:id", chatHandler.DeleteChatRoom)
	e.GET("/list/:id", chatHandler.GetChatMsgListByRoomID)
	e.DELETE("/all/:id", chatHandler.DeleteChatByRoomID)
	e.GET("/character/name/:id", chatHandler.GetCharacterNameByRoomID)

	return e
}
