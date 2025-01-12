package main

import (
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
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

func (app *Config) profileHandler(w http.ResponseWriter, r *http.Request) {
	// Extract query parameters
	gender := r.URL.Query().Get("gender")
	characterID := r.URL.Query().Get("character_id")

	if gender == "" || characterID == "" {
		http.Error(w, "Missing query parameters", http.StatusBadRequest)
		return
	}

	imageBaseDir := "/app/images"

	// Construct the file path
	filePath := filepath.Join(imageBaseDir, "profile_"+gender+"_"+characterID+".png")

	// Serve the file
	http.ServeFile(w, r, filePath)
}
