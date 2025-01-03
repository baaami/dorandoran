package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func (app *Config) routes() http.Handler {
	mux := chi.NewRouter()

	// CORS 설정
	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// 채팅방 리스트 획득
	mux.Get("/room/list", app.getChatRoomList)

	// 채팅방 상세 정보 획득
	mux.Get("/room/{id}", app.getChatRoomByID)

	mux.Post("/room/join", app.handleRoomJoin)
	mux.Delete("/room/leave/{id}", app.leaveChatRoom)

	// TODO: Timeout된 채팅방들은 삭제해주는 고루틴이 필요함
	mux.Delete("/room/delete/{id}", app.deleteChatRoom)

	// 채팅 메시지 추가
	mux.Post("/msg", app.addChatMsg)

	// 채팅 메시지 읽음 처리 추가
	mux.Post("/msg/read", app.handleChatRead)

	// 채팅 내역 조회
	mux.Get("/list/{id}", app.getChatMsgListByRoomID) // by roomid

	// 채팅 내역 삭제
	mux.Delete("/all/{id}", app.deleteChatByRoomID) // by roomid

	// 게임방 내 캐릭터명 조회
	mux.Get("/character/name/{id}", app.getChatMsgListByRoomID) // by roomid

	return mux
}
