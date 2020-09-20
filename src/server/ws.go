package server

import (
	"context"
	"github.com/gorilla/websocket"
	"log"
	"speech-to-text-back/src/Speech2Text"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 1024 * 10
)

var newline = []byte{'\n'}

func initWs(conn *websocket.Conn) {
	conn.SetReadLimit(maxMessageSize)
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
}

func listen(conn *websocket.Conn, fileBuffer chan []byte) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		fileBuffer <- message
	}
}

func sendResp(conn *websocket.Conn, streamResp chan []byte) {
	ticker := time.NewTicker(pingPeriod)
	defer conn.Close()
	for {
		select {
		case message := <-streamResp:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			w, err := conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(streamResp)
			for i := 0; i < n; i++ {
				_, _ = w.Write(newline)
				_, _ = w.Write(<-streamResp)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func streamS2t(conn *websocket.Conn) {
	defer conn.Close()
	initWs(conn)
	ctx := context.Background()

	fileBuffer := make(chan []byte)

	s := Speech2Text.NewStream(ctx, fileBuffer)
	s.Start()
	go listen(conn, fileBuffer)
	go sendResp(conn, s.StreamResp)
}
