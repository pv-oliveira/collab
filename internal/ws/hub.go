package ws

import "sync"

type Message struct {
	DocumentID string `json:"document_id"`
	UserID     string `json:"user_id"`
	Type       string `json:"type"` // "edit"
	Content    string `json:"content"`
}
type Hub struct {
	clients    map[string]map[*WebsocketClient]bool
	register   chan *WebsocketClient
	unregister chan *WebsocketClient
	broadcast  chan Message
	mu         sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*WebsocketClient]bool),
		register:   make(chan *WebsocketClient),
		unregister: make(chan *WebsocketClient),
		broadcast:  make(chan Message),
	}
}

func (h *Hub) Run() {
	for {
		select {

		case client := <-h.register:
			if _, ok := h.clients[client.documentID]; !ok {
				h.clients[client.documentID] = make(map[*WebsocketClient]bool)
			}
			h.clients[client.documentID][client] = true

		case client := <-h.unregister:
			if clients, ok := h.clients[client.documentID]; ok {
				if _, exists := clients[client]; exists {
					delete(clients, client)
					close(client.send)
				}
			}

		case msg := <-h.broadcast:
			if clients, ok := h.clients[msg.DocumentID]; ok {
				for client := range clients {

					// não manda pra quem enviou
					if client.userID == msg.UserID {
						continue
					}

					select {
					case client.send <- []byte(msg.Content):
					default:
						close(client.send)
						delete(clients, client)
					}
				}
			}
		}
	}
}
