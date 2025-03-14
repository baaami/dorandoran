package handler

import (
	"log"
	"net/http"
	"solo/pkg/models"
	"solo/pkg/types/commontype"
	"solo/services/auth/service"
	"strings"

	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) KakaoLoginHandler(c echo.Context) error {
	var requestData struct {
		AccessToken string `json:"accessToken"`
	}

	if err := c.Bind(&requestData); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	var snsID string
	if strings.HasPrefix(requestData.AccessToken, "masterkey-") {
		parts := strings.Split(requestData.AccessToken, "-")
		if len(parts) == 2 {
			snsID = parts[1]
		} else {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid masterkey format"})
		}
	} else {
		var err error
		snsID, err = h.authService.VerifyKakaoAccessToken(requestData.AccessToken)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
		}
	}

	// user-service에서 유저 조회
	loginUser, err := h.authService.GetExistUserByUserSrv(commontype.KAKAO, snsID)
	if err != nil {
		log.Printf("Error checking user existence: %v\n", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	// 유저가 존재하지 않는 경우 회원가입
	if loginUser == (models.User{}) {
		loginUser, err = h.authService.RegisterNewUser(commontype.KAKAO, snsID)
		if err != nil {
			log.Printf("Failed to register new user, err: %s", err.Error())
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to register new user"})
		}
	}

	// Redis에 세션 생성 (authService의 CreateSession API 사용)
	sessionID := h.authService.CreateSession(loginUser.ID)

	SetSessionCookie(c, sessionID)
	return c.JSON(http.StatusOK, loginUser)
}

func (h *AuthHandler) NaverLoginHandler(c echo.Context) error {
	var requestData struct {
		AccessToken string `json:"accessToken"`
	}

	if err := c.Bind(&requestData); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	var snsID string
	if strings.HasPrefix(requestData.AccessToken, "masterkey-") {
		parts := strings.Split(requestData.AccessToken, "-")
		if len(parts) == 2 {
			snsID = parts[1]
		} else {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid masterkey format"})
		}
	} else {
		var err error
		snsID, err = h.authService.VerifyNaverAccessToken(requestData.AccessToken)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
		}
	}

	// user-service에서 유저 조회
	loginUser, err := h.authService.GetExistUserByUserSrv(commontype.NAVER, snsID)
	if err != nil {
		log.Printf("Error checking user existence: %v\n", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	// 유저가 존재하지 않는 경우 회원가입
	if loginUser == (models.User{}) {
		loginUser, err = h.authService.RegisterNewUser(commontype.NAVER, snsID)
		if err != nil {
			log.Printf("Failed to register new user, err: %s", err.Error())
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to register new user"})
		}
	}

	// Redis에 세션 생성 (authService의 CreateSession API 사용)
	sessionID := h.authService.CreateSession(loginUser.ID)

	SetSessionCookie(c, sessionID)
	return c.JSON(http.StatusOK, loginUser)
}

// 세션 쿠키 설정
func SetSessionCookie(c echo.Context, sessionID string) {
	cookie := new(http.Cookie)
	cookie.Name = "session_id"
	cookie.Value = sessionID
	cookie.HttpOnly = true
	cookie.Secure = true
	cookie.Path = "/"
	c.SetCookie(cookie)
}
