package main

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write the file to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	websocker = &WebSocker{
		log.New(os.Stderr, "WSOCK   ", log.LstdFlags),
	}
)

type WebSocker struct {
	log *log.Logger
}

func (ws *WebSocker) handle(w http.ResponseWriter, req *http.Request, user string, tmpl *template.Template) {
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			ws.log.Println(err)
		}
		return
	}

	go ws.writer(conn, user, tmpl)
	ws.reader(conn)
}

func (ws *WebSocker) writer(conn *websocket.Conn, user string, tmpl *template.Template) {
	pingTicker := time.NewTicker(pingPeriod)
	chl, chq := notifier.listen(user)
	ws.log.Println(user, "connected")
	defer func() {
		pingTicker.Stop()
		conn.Close()
		close(chq)
		ws.log.Println(user, "disconnected")
	}()
	for {
		var buf bytes.Buffer
		select {
		case p := <-chl:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			buf.Reset()
			if err := tmpl.Execute(&buf, p); err != nil {
				ws.log.Println(err)
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, buf.Bytes()); err != nil {
				return
			}
			ws.log.Println("notified:", user)
		case <-pingTicker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				ws.log.Println("ping:", err)
				return
			}
		}
	}
}

func (ws *WebSocker) reader(conn *websocket.Conn) {
	defer conn.Close()
	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
