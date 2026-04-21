package ws

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WSHandler struct {
	hub *Hub
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *WSHandler) Handle(c *gin.Context) {
	userID := c.GetString("userID")
	docID := c.Param("id")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	client := &WebsocketClient{
		userID:     userID,
		documentID: docID,
		conn:       conn,
		send:       make(chan []byte, 256),
		hub:        h.hub,
	}

	h.hub.register <- client
	go client.WritePump()
	go client.ReadPump()
}
