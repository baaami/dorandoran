package handler

import (
	"io"
	"log"
	"net/http"
	"path/filepath"
	"solo/pkg/helper"

	"github.com/labstack/echo/v4"
)

type GatewayHandler struct {
}

func NewGatewayHandler() *GatewayHandler {
	return &GatewayHandler{}
}

// ProxyService - API를 프록시해주는 역할
func (h *GatewayHandler) ProxyService(c echo.Context) error {
	log.Printf("Proxy Request URL: %s", c.Request().URL)

	// 요청 경로에서 첫 번째 경로 요소를 추출
	firstPath, trimmedPath := helper.ExtractFirstPath(c.Request().URL.Path)

	// 첫 번째 경로 요소에 따라 targetURL 설정
	baseURL := "http://" + "doran-" + firstPath
	targetURL := baseURL + trimmedPath

	// 쿼리 스트링 추가
	if c.QueryString() != "" {
		targetURL += "?" + c.QueryString()
	}

	// 새로운 요청 생성 (전달받은 HTTP 메서드 유지)
	req, err := http.NewRequest(c.Request().Method, targetURL, c.Request().Body)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create request"})
	}

	// 원본 요청 헤더 복사
	for key, values := range c.Request().Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// HTTP 클라이언트 생성 및 요청 전송
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send request: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to send request"})
	}
	defer resp.Body.Close()

	// 응답 헤더 복사
	for key, values := range resp.Header {
		for _, value := range values {
			c.Response().Header().Add(key, value)
		}
	}

	// 상태 코드 설정
	c.Response().WriteHeader(resp.StatusCode)

	// 응답 본문을 클라이언트에게 전달
	_, err = io.Copy(c.Response().Writer, resp.Body)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to copy response body"})
	}

	return nil
}

// ProfileHandler - 프로필 이미지 제공
func (h *GatewayHandler) ProfileHandler(c echo.Context) error {
	// Extract query parameters
	gender := c.QueryParam("gender")
	characterID := c.QueryParam("character_id")

	if gender == "" || characterID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing query parameters"})
	}

	imageBaseDir := "/app/images"
	filePath := filepath.Join(imageBaseDir, "profile_"+gender+"_"+characterID+".png")

	return c.File(filePath)
}
