package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/net/websocket"
)

type Server struct {
	chatrooms map[string]map[*websocket.Conn]bool
}

func NewServer() *Server {
	return &Server{
		chatrooms: make(map[string]map[*websocket.Conn]bool),
	}
}

func (s *Server) handleWS(ws *websocket.Conn) {
	chatroom := "general"
	if strings.Contains(ws.Request().URL.RawQuery, "chatroom=") {
		params := strings.Split(ws.Request().URL.RawQuery, "=")
		if len(params) > 1 {
			chatroom = params[1]
		}
	}

	fmt.Println("New connection in chatroom:", chatroom, "from:", ws.RemoteAddr())
	if s.chatrooms[chatroom] == nil {
		s.chatrooms[chatroom] = make(map[*websocket.Conn]bool)
	}
	s.chatrooms[chatroom][ws] = true
	s.readLoop(ws, chatroom)
	delete(s.chatrooms[chatroom], ws)
	ws.Close()
	fmt.Println("Client disconnected from", chatroom)
}

func (s *Server) readLoop(ws *websocket.Conn, chatroom string) {
	buff := make([]byte, 1024)
	for {
		n, err := ws.Read(buff)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println("Read error:", err)
			continue
		}

		msg := string(buff[:n])
		fmt.Println("Received message in", chatroom, ":", msg)
		messagePayload, _ := json.Marshal(map[string]string{
			"chatroom": chatroom,
			"message":  msg,
		})
		s.broadcast(chatroom, messagePayload)
	}
}

func (s *Server) broadcast(chatroom string, msg []byte) {
	for ws := range s.chatrooms[chatroom] {
		go func(ws *websocket.Conn) {
			if _, err := ws.Write(msg); err != nil {
				fmt.Println("Write error:", err)
				ws.Close()
				delete(s.chatrooms[chatroom], ws)
			}
		}(ws)
	}
}

func main() {
	server := NewServer()
	http.Handle("/ws", websocket.Handler(server.handleWS))
	http.Handle("/", http.FileServer(http.Dir("./frontend")))
	http.ListenAndServe(":3000", nil)
}
