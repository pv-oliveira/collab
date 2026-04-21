package ws

import (
	"github.com/gorilla/websocket"
)

type WebsocketClient struct {
	conn       *websocket.Conn
	send       chan []byte
	userID     string
	documentID string
	hub        *Hub
}

func (c *WebsocketClient) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		message := Message{
			DocumentID: c.documentID,
			UserID:     c.userID,
			Type:       "edit",
			Content:    string(msg),
		}

		c.hub.broadcast <- message
	}
}

func (c *WebsocketClient) WritePump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for msg := range c.send {
		err := c.conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			return
		}
	}
}
