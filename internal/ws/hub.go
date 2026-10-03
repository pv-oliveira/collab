package ws

import (
	"context"
	"log"
)

type Event struct {
	ID         string `json:"id"` // único por edição: chave do lock do autosave
	DocumentID string `json:"document_id"`
	UserID     string `json:"user_id"`
	SenderID   string `json:"sender_id"` // id da conexão que enviou
	Type       string `json:"type"`
	Payload    string `json:"payload"`
	Timestamp  int64  `json:"timestamp"`
}

// Hub guarda as conexões desta instância. Só a goroutine do Run toca no mapa
// clients; as outras goroutines conversam com ela pelos canais.
type Hub struct {
	clients    map[string]map[*WebsocketClient]bool
	register   chan *WebsocketClient
	unregister chan *WebsocketClient
	broadcast  chan Event

	bus *RedisBus
}

func NewHub(bus *RedisBus) *Hub {
	return &Hub{
		clients:    make(map[string]map[*WebsocketClient]bool),
		register:   make(chan *WebsocketClient),
		unregister: make(chan *WebsocketClient),
		broadcast:  make(chan Event),
		bus:        bus,
	}
}

// Publish envia o evento só para o Redis. A entrega acontece quando ele volta
// pela inscrição, inclusive nesta instância: um único caminho, sem duplicar.
func (h *Hub) Publish(ctx context.Context, event Event) {
	if err := h.bus.Publish(ctx, event); err != nil {
		log.Println("ws: falha ao publicar no redis:", err)
	}
}

func (h *Hub) Run(ctx context.Context) error {
	err := h.bus.Subscribe(ctx, func(event Event) {
		h.broadcast <- event
	})
	if err != nil {
		return err
	}

	for {
		select {
		case client := <-h.register:
			if _, ok := h.clients[client.documentID]; !ok {
				h.clients[client.documentID] = make(map[*WebsocketClient]bool)
			}
			h.clients[client.documentID][client] = true

		case client := <-h.unregister:
			// ReadPump e WritePump chamam unregister: só fecha na primeira vez.
			if clients, ok := h.clients[client.documentID]; ok {
				if _, exists := clients[client]; exists {
					delete(clients, client)
					close(client.send)
				}
			}

		case event := <-h.broadcast:
			h.broadcastLocal(event)
		}
	}
}

func (h *Hub) broadcastLocal(event Event) {
	clients := h.clients[event.DocumentID]

	for client := range clients {
		// não manda de volta para a conexão que enviou
		if client.id == event.SenderID {
			continue
		}

		select {
		case client.send <- []byte(event.Payload):
		default:
			// cliente lento: buffer cheio, desconecta
			close(client.send)
			delete(clients, client)
		}
	}
}
