package handler

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"solo/pkg/dto"
	"solo/pkg/models"
	"solo/pkg/types/commontype"
	"solo/services/chat/service"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	var roomlist []dto.RoomListResponse
	for _, room := range rooms {
		latestMessage, err := h.chatService.GetLatestMessage(room.ID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve latest message"})
		}

		unreadCount, err := h.chatService.GetUnreadCount(room.ID, userID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve unread count"})
		}

		gamerInfo, err := h.chatService.GetGamerInfo(userID, room.ID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve gamer info"})
		}
		if gamerInfo == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Gamer info not found"})
		}

		roomlist = append(roomlist, dto.RoomListResponse{
			ID:       room.ID,
			RoomName: room.Name,
			RoomType: room.Type,
			LastMessage: dto.LastMessage{
				SenderID: latestMessage.SenderID,
				Message:  latestMessage.Message,
				GameInfo: commontype.GameInfo{
					CharacterID:        gamerInfo.CharacterID,
					CharacterName:      gamerInfo.CharacterName,
					CharacterAvatarURL: gamerInfo.CharacterAvatarURL,
				},
				CreatedAt: latestMessage.CreatedAt,
			},
			UnreadCount: unreadCount,
			CreatedAt:   room.CreatedAt,
			ModifiedAt:  room.ModifiedAt,
		})
	}

	return c.JSON(http.StatusOK, roomlist)
}

// 특정 채팅방 상세 정보 조회
func (h *ChatHandler) GetChatRoomByID(c echo.Context) error {
	roomID := c.Param("id")

	room, err := h.chatService.GetChatRoomDetailByID(roomID)
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

// 특정 채팅방의 메시지 목록 조회
func (h *ChatHandler) GetChatMsgListByRoomID(c echo.Context) error {
	roomID := c.Param("id")

	pageStr := c.QueryParam("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	messages, totalCount, err := h.chatService.GetChatMsgListByRoomID(roomID, page, commontype.DEFAULT_PAGE_SIZE)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch chat messages"})
	}

	response := dto.ChatListResponse{
		Data:        messages,
		CurrentPage: page,
		NextPage:    page + 1,
		HasNextPage: page < int(totalCount),
		TotalPages:  int(math.Ceil(float64(totalCount) / float64(commontype.DEFAULT_PAGE_SIZE))),
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

// 밸런스 게임 조회
func (h *ChatHandler) GetBalanceFormByID(c echo.Context) error {
	formID := c.Param("formid")
	objectID, err := primitive.ObjectIDFromHex(formID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid form ID format"})
	}

	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	form, err := h.chatService.GetBalanceForm(objectID, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve balance form"})
	}

	return c.JSON(http.StatusOK, form)
}

// 밸런스 게임 투표 삽입
func (h *ChatHandler) InsertBalanceFormVote(c echo.Context) error {
	formID := c.Param("formid")
	objectID, err := primitive.ObjectIDFromHex(formID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid form ID format"})
	}

	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	var voteDTO dto.BalanceFormVoteDTO
	if err := c.Bind(&voteDTO); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	vote := models.BalanceFormVote{
		FormID:    objectID,
		UserID:    userID,
		Choiced:   voteDTO.Choiced,
		CreatedAt: time.Now(),
	}

	err = h.chatService.InsertBalanceFormVote(&vote)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to insert balance form vote"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Balance form vote inserted successfully"})
}

// 밸런스 게임 투표 취소
func (h *ChatHandler) CancelBalanceFormVote(c echo.Context) error {
	formID := c.Param("formid")
	objectID, err := primitive.ObjectIDFromHex(formID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid form ID format"})
	}

	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	err = h.chatService.CancelBalanceFormVote(objectID, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to cancel balance form vote"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Balance form vote canceled successfully"})
}

// 밸런스 게임 투표 댓글 삽입
func (h *ChatHandler) InsertBalanceFormComment(c echo.Context) error {
	formID := c.Param("formid")
	objectID, err := primitive.ObjectIDFromHex(formID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid form ID format"})
	}

	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	var commentDTO dto.BalanceFormCommentDTO
	if err := c.Bind(&commentDTO); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	comment := models.BalanceFormComment{
		FormID:    objectID,
		SenderID:  userID,
		Message:   commentDTO.Comment,
		CreatedAt: time.Now(),
	}

	err = h.chatService.InsertBalanceFormComment(objectID, &comment)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to insert balance form comment"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Balance form comment inserted successfully"})
}

// 밸런스 게임 투표 댓글 조회
func (h *ChatHandler) GetBalanceFormComments(c echo.Context) error {
	formID := c.Param("formid")
	objectID, err := primitive.ObjectIDFromHex(formID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid form ID format"})
	}

	pageStr := c.QueryParam("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	comments, totalCount, err := h.chatService.GetBalanceFormComments(objectID, page, commontype.DEFAULT_PAGE_SIZE)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve balance form comments"})
	}

	response := dto.BalanceFormCommentListResponse{
		Data:        comments,
		CurrentPage: page,
		NextPage:    page + 1,
		HasNextPage: page < int(totalCount),
		TotalPages:  int(math.Ceil(float64(totalCount) / float64(commontype.DEFAULT_PAGE_SIZE))),
	}

	return c.JSON(http.StatusOK, response)
}
