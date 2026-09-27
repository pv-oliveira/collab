package ws

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
)

type WebsocketClient struct {
	id         string
	conn       *websocket.Conn
	send       chan []byte
	userID     string
	documentID string
	hub        *Hub
}

func (c *WebsocketClient) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close() // conexão já está encerrando; erro aqui não muda nada
	}()

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		c.hub.Publish(context.Background(), Event{
			DocumentID: c.documentID,
			UserID:     c.userID,
			SenderID:   c.id,
			Type:       "edit",
			Payload:    string(msg),
			Timestamp:  time.Now().Unix(),
		})
	}
}

func (c *WebsocketClient) WritePump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()

	for msg := range c.send {
		err := c.conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			return
		}
	}
}
