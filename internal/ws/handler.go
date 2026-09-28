package ws

import (
	"apis/internal/middleware"
	"apis/internal/services"
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// O navegador não permite headers customizados no WebSocket, então o JWT vem
// no subprotocolo: new WebSocket(url, ["access_token", jwt]).
const tokenProtocol = "access_token"

type WSHandler struct {
	hub       *Hub
	docs      *services.DocumentService
	jwtSecret string
	upgrader  websocket.Upgrader
}

func NewWSHandler(hub *Hub, docs *services.DocumentService, jwtSecret string, allowedOrigins []string) *WSHandler {
	return &WSHandler{
		hub:       hub,
		docs:      docs,
		jwtSecret: jwtSecret,
		upgrader: websocket.Upgrader{
			// O servidor precisa devolver um dos subprotocolos pedidos, senão o navegador fecha a conexão.
			Subprotocols: []string{tokenProtocol},
			CheckOrigin:  checkOrigin(allowedOrigins),
		},
	}
}

// checkOrigin protege contra Cross-Site WebSocket Hijacking: um navegador
// só conecta a partir de uma origem listada. Clientes que não são navegador
// (sem header Origin) passam, pois o ataque só existe via navegador.
func checkOrigin(allowed []string) func(*http.Request) bool {
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == "" || middleware.OriginAllowed(origin, allowed)
	}
}

func (h *WSHandler) Handle(c *gin.Context) {
	userID, err := middleware.ParseToken(h.jwtSecret, tokenFromSubprotocol(c.Request))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// Só o dono do documento entra na sala (checado antes do upgrade).
	docID := c.Param("id")
	if _, err := h.docs.GetByID(userID, docID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("ws: upgrade:", err)
		return
	}

	client := &WebsocketClient{
		id:         uuid.NewString(),
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

// tokenFromSubprotocol lê ["access_token", "<jwt>"] do header Sec-WebSocket-Protocol.
func tokenFromSubprotocol(r *http.Request) string {
	protocols := websocket.Subprotocols(r)
	for i := 0; i < len(protocols)-1; i++ {
		if protocols[i] == tokenProtocol {
			return protocols[i+1]
		}
	}
	return ""
}
