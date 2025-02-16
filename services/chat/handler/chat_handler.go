package handler

import (
	"net/http"
	"strconv"

	"solo/pkg/types/commontype"
	"solo/services/chat/service"

	"github.com/labstack/echo/v4"
)

type ChatHandler struct {
	chatService *service.ChatService
}

func NewChatHandler(chatService *service.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

// X-User-ID 헤더에서 유저 ID를 가져오는 유틸 함수
func getUserID(c echo.Context) (int, error) {
	userIDStr := c.Request().Header.Get("X-User-ID")
	if userIDStr == "" {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, "User ID is required")
	}
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, "Invalid User ID format")
	}
	return userID, nil
}

// 특정 유저의 채팅방 목록 조회
func (h *ChatHandler) GetChatRoomList(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	rooms, err := h.chatService.GetChatRoomList(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve chat rooms"})
	}

	return c.JSON(http.StatusOK, rooms)
}

// 특정 채팅방 상세 정보 조회
func (h *ChatHandler) GetChatRoomByID(c echo.Context) error {
	roomID := c.Param("id")

	room, err := h.chatService.GetChatRoomByID(roomID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to find chat room"})
	}
	if room == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Chat room not found"})
	}

	return c.JSON(http.StatusOK, room)
}

// 채팅방 삭제
func (h *ChatHandler) DeleteChatRoom(c echo.Context) error {
	roomID := c.Param("id")

	err := h.chatService.DeleteChatRoom(roomID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete chat room"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Chat room deleted successfully"})
}

// 채팅 메시지 추가
func (h *ChatHandler) AddChatMsg(c echo.Context) error {
	var chatMsg commontype.Chat
	if err := c.Bind(&chatMsg); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	messageID, err := h.chatService.AddChatMsg(chatMsg)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to insert chat message"})
	}

	return c.JSON(http.StatusCreated, map[string]string{"message": "Chat message inserted successfully", "message_id": messageID.Hex()})
}

// 특정 채팅방의 메시지 목록 조회
func (h *ChatHandler) GetChatMsgListByRoomID(c echo.Context) error {
	roomID := c.Param("id")

	pageStr := c.QueryParam("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	messages, totalCount, err := h.chatService.GetChatMsgListByRoomID(roomID, page, 20)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch chat messages"})
	}

	response := map[string]interface{}{
		"messages":    messages,
		"total_count": totalCount,
	}

	return c.JSON(http.StatusOK, response)
}

// 특정 채팅방의 메시지 삭제
func (h *ChatHandler) DeleteChatByRoomID(c echo.Context) error {
	roomID := c.Param("id")

	err := h.chatService.DeleteChatByRoomID(roomID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete chat messages"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Chat messages deleted successfully"})
}

// 특정 유저의 게임 캐릭터 정보 조회
func (h *ChatHandler) GetCharacterNameByRoomID(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	roomID := c.Param("id")

	gamerInfo, err := h.chatService.GetCharacterNameByRoomID(userID, roomID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve character name"})
	}
	if gamerInfo == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Character info not found"})
	}

	return c.JSON(http.StatusOK, gamerInfo)
}
