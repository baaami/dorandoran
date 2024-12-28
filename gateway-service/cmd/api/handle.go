package main

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

func (app *Config) usage(w http.ResponseWriter, r *http.Request) {
	payload := jsonResponse{
		Error:   false,
		Message: "I will Write Usage Here",
	}

	_ = app.writeJSON(w, http.StatusOK, payload)
}

// API를 프록시해주는 역할
func (app *Config) proxyService() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Proxy Request URL: %s", r.URL)

		// 요청 경로에서 첫 번째 경로 요소를 추출
		firstPath, trimmedPath := extractFirstPath(r.URL.Path)

		// 첫 번째 경로 요소에 따라 targetURL 설정
		baseURL := "http://" + firstPath + "-service"
		targetURL := baseURL + trimmedPath

		// 쿼리 스트링이 존재하면 targetURL에 추가
		if r.URL.RawQuery != "" {
			targetURL = targetURL + "?" + r.URL.RawQuery
		}

		// 송신할 요청 생성
		req, err := http.NewRequest(r.Method, targetURL, r.Body)
		if err != nil {
			log.Printf("method: %s, url: %s, body: %v, err: %s", r.Method, targetURL, r.Body, err.Error())
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
		}

		// 원본 요청의 헤더를 모두 복사
		for name, values := range r.Header {
			for _, value := range values {
				req.Header.Add(name, value)
			}
		}

		// HTTP 클라이언트 생성
		client := &http.Client{}

		// 요청을 송신하고 응답을 받음
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "Failed to send request", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// 응답 헤더 설정
		for name, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(name, value)
			}
		}

		// 상태 코드 설정
		w.WriteHeader(resp.StatusCode)

		// 응답 본문을 클라이언트에게 전달
		_, err = io.Copy(w, resp.Body)
		if err != nil {
			http.Error(w, "Failed to copy response body", http.StatusInternalServerError)
			return
		}
	}
}

// 첫 번째 경로 요소를 추출하고 나머지 경로를 반환하는 함수
func extractFirstPath(path string) (string, string) {
	// 경로를 '/'로 분리
	parts := strings.SplitN(path, "/", 3)

	// 첫 번째 경로 요소 (예: "user")와 나머지 경로 반환
	if len(parts) > 1 {
		firstPath := parts[1] // 첫 번째 경로 요소 (예: "user")
		if len(parts) > 2 {
			return firstPath, "/" + parts[2] // 나머지 경로 (예: "/read")
		}
		return firstPath, "/" // 경로가 없을 경우 루트 경로 반환
	}

	// 경로가 비었을 경우 기본값 반환
	return "", "/"
}

func (app *Config) proxySocketServer(targetBaseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// WebSocket 업그레이드 요청인지 확인
		if !websocket.IsWebSocketUpgrade(r) {
			http.Error(w, "Invalid WebSocket upgrade request", http.StatusBadRequest)
			return
		}

		// WebSocket 대상 서버 주소
		targetURL := targetBaseURL + r.URL.Path

		// WebSocket Dialer 생성
		dialer := websocket.DefaultDialer

		userIDStr := r.Header.Get("X-User-ID")

		// WebSocket 요청에 필요한 헤더만 생성
		requestHeaders := http.Header{}
		requestHeaders.Set("Sec-WebSocket-Protocol", r.Header.Get("Sec-WebSocket-Protocol"))
		requestHeaders.Set("Origin", r.Header.Get("Origin"))
		requestHeaders.Set("X-User-ID", userIDStr)

		// WebSocket 서버로 업그레이드 요청 전달
		targetConn, _, err := dialer.Dial(targetURL, requestHeaders)
		if err != nil {
			log.Printf("Failed to connect to WebSocket server: %v", err)
			http.Error(w, "Failed to connect to WebSocket server", http.StatusBadGateway)
			return
		}
		defer targetConn.Close()

		// 클라이언트 WebSocket 업그레이드
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}

		clientConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Failed to upgrade WebSocket connection: %v", err)
			return
		}
		defer clientConn.Close()

		// 메시지 중계
		go forwardMessages(clientConn, targetConn) // 클라이언트 → 서버
		forwardMessages(targetConn, clientConn)    // 서버 → 클라이언트
	}
}

func forwardMessages(src, dest *websocket.Conn) {
	for {
		// 메시지 읽기
		messageType, msg, err := src.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				log.Printf("Unexpected WebSocket close error")
			} else {
				log.Printf("WebSocket connection closed by client")
			}
			break
		}

		// 메시지 쓰기
		err = dest.WriteMessage(messageType, msg)
		if err != nil {
			log.Printf("Error forwarding WebSocket message: %v", err)
			break
		}
	}
}
