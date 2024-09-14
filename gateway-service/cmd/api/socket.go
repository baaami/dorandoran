package main

import (
	"fmt"
	"log"

	socketio "github.com/googollee/go-socket.io"
)

func (app *Config) InitSocket() {
	app.ws = socketio.NewServer(nil)

	app.ws.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		fmt.Println("connected to root namespace:", s.ID())
		return nil
	})

	app.ws.OnConnect("/chat", func(s socketio.Conn) error {
		s.SetContext("")
		fmt.Println("connected to chat namespace:", s.ID())
		return nil
	})

	app.ws.OnEvent("/", "notice", func(s socketio.Conn, msg string) {
		fmt.Println("notice:", msg)
		s.Emit("reply", "have "+msg)
	})

	app.ws.OnEvent("/chat", "message", func(s socketio.Conn, msg string) string {
		fmt.Println("chat message:", msg)
		s.Emit("reply", "received "+msg)
		return "received " + msg
	})

	app.ws.OnEvent("/", "bye", func(s socketio.Conn) string {
		last := s.Context().(string)
		s.Emit("bye", last)
		s.Close()
		return last
	})

	app.ws.OnError("/", func(s socketio.Conn, e error) {
		fmt.Printf("Error on root namespace for client %s: %v\n", s.ID(), e)
	})

	app.ws.OnError("/chat", func(s socketio.Conn, e error) {
		fmt.Printf("Error on chat namespace for client %s: %v\n", s.ID(), e)
	})

	app.ws.OnDisconnect("/", func(s socketio.Conn, reason string) {
		fmt.Printf("Client %s disconnected from root: %s\n", s.ID(), reason)
	})

	app.ws.OnDisconnect("/chat", func(s socketio.Conn, reason string) {
		fmt.Printf("Client %s disconnected from chat: %s\n", s.ID(), reason)
	})

	go func() {
		if err := app.ws.Serve(); err != nil {
			log.Fatalf("Socket.IO server error: %v", err)
		}
	}()
}
