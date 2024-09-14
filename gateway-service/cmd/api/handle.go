package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type UserPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func (app *Config) usage(w http.ResponseWriter, r *http.Request) {
	payload := jsonResponse{
		Error:   false,
		Message: "I will Write Usage Here",
	}

	_ = app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) userService(w http.ResponseWriter, r *http.Request) {
	// 요청에서 데이터를 읽음 (예: JSON 데이터)
	var userData map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&userData)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// 데이터를 송신할 URL
	targetURL := "http://user-service"

	// 데이터를 JSON 형식으로 변환
	jsonData, err := json.Marshal(userData)
	if err != nil {
		http.Error(w, "Error converting data to JSON", http.StatusInternalServerError)
		return
	}

	// 송신할 요청 생성
	req, err := http.NewRequest(r.Method, targetURL, bytes.NewBuffer(jsonData))
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// 요청에 필요한 헤더 추가
	req.Header.Set("Content-Type", "application/json")

	// HTTP 클라이언트 생성
	client := &http.Client{}

	// 요청을 송신하고 응답을 받음
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to send request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// 응답 내용 읽기
	body, err := io.ReadAll(resp.Body) // ioutil.ReadAll 대신 io.ReadAll 사용
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	// 응답을 클라이언트에게 반환
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func (app *Config) chatService(w http.ResponseWriter, r *http.Request) {
	payload := jsonResponse{
		Error:   false,
		Message: "I will connect chat service here",
	}

	_ = app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) authService(w http.ResponseWriter, r *http.Request) {
	payload := jsonResponse{
		Error:   false,
		Message: "I will Write Usage Here",
	}

	_ = app.writeJSON(w, http.StatusOK, payload)
}
