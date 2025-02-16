package transport

import (
	"solo/services/chat/handler"

	"github.com/labstack/echo/v4"
)

func NewRouter(chatHandler *handler.ChatHandler) *echo.Echo {
	e := echo.New()

	// 채팅방 관련 라우팅
	e.GET("/room/list", chatHandler.GetChatRoomList)         // 채팅방 목록 조회
	e.GET("/room/:id", chatHandler.GetChatRoomByID)          // 채팅방 상세 조회
	e.DELETE("/room/delete/:id", chatHandler.DeleteChatRoom) // 채팅방 삭제

	// 채팅 메시지 관련 라우팅
	e.POST("/msg", chatHandler.AddChatMsg)                 // 채팅 메시지 추가
	e.GET("/list/:id", chatHandler.GetChatMsgListByRoomID) // 특정 방의 채팅 내역 조회
	e.DELETE("/all/:id", chatHandler.DeleteChatByRoomID)   // 특정 방의 모든 채팅 삭제

	// 게임 캐릭터 정보 조회
	e.GET("/character/name/:id", chatHandler.GetCharacterNameByRoomID) // 특정 방의 캐릭터 정보 조회

	return e
}
