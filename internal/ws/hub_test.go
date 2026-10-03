package ws

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// startHubs sobe n instâncias do hub ligadas ao mesmo Redis em memória,
// simulando n réplicas da API.
func startHubs(t *testing.T, n int) []*Hub {
	t.Helper()
	mr := miniredis.RunT(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	hubs := make([]*Hub, n)
	for i := range hubs {
		// Um client por hub, como réplicas separadas.
		hubs[i] = NewHub(NewRedisBus(redis.NewClient(&redis.Options{Addr: mr.Addr()})))
		go func(h *Hub) { _ = h.Run(ctx) }(hubs[i])
	}
	// Run inscreve no Redis antes de entrar no loop; register só é lido depois disso.
	return hubs
}

func newTestClient(h *Hub, id, userID, docID string, buffer int) *WebsocketClient {
	c := &WebsocketClient{id: id, userID: userID, documentID: docID, send: make(chan []byte, buffer), hub: h}
	h.register <- c
	return c
}

// received coleta o que chegar em c.send dentro de uma janela curta.
func received(c *WebsocketClient) []string {
	var got []string
	timeout := time.After(200 * time.Millisecond)
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return got
			}
			got = append(got, string(msg))
		case <-timeout:
			return got
		}
	}
}

func TestHubDeliversAcrossInstances(t *testing.T) {
	hubs := startHubs(t, 2)
	a, b := hubs[0], hubs[1]

	sender := newTestClient(a, "a1", "u1", "doc1", 8)
	sameUserOtherTab := newTestClient(a, "a2", "u1", "doc1", 8)
	otherInstance := newTestClient(b, "b1", "u2", "doc1", 8)
	otherDocument := newTestClient(b, "b2", "u2", "doc2", 8)

	a.Publish(context.Background(), Event{DocumentID: "doc1", UserID: "u1", SenderID: "a1", Payload: "oi"})

	if got := received(sender); len(got) != 0 {
		t.Errorf("remetente recebeu o próprio eco: %v", got)
	}
	// Exatamente 1: pega tanto o canal "*" literal (0) quanto a entrega duplicada (2).
	if got := received(sameUserOtherTab); len(got) != 1 {
		t.Errorf("outra aba do mesmo usuário: esperava 1 mensagem, recebeu %v", got)
	}
	if got := received(otherInstance); len(got) != 1 || got[0] != "oi" {
		t.Errorf("outra instância: esperava [oi], recebeu %v", got)
	}
	if got := received(otherDocument); len(got) != 0 {
		t.Errorf("cliente de outro documento recebeu: %v", got)
	}
}

func TestHubUnregisterTwiceDoesNotPanic(t *testing.T) {
	h := startHubs(t, 1)[0]
	c := newTestClient(h, "c1", "u1", "doc1", 8)

	// ReadPump e WritePump chamam unregister ao sair; fechar o canal duas vezes daria panic.
	h.unregister <- c
	h.unregister <- c

	if _, ok := <-c.send; ok {
		t.Error("canal send deveria estar fechado")
	}
}

func TestHubDisconnectsSlowClientWithoutBlocking(t *testing.T) {
	h := startHubs(t, 1)[0]
	slow := newTestClient(h, "slow", "u1", "doc1", 1) // buffer de 1 e ninguém lê
	fast := newTestClient(h, "fast", "u2", "doc1", 8)

	for _, p := range []string{"1", "2", "3"} {
		h.Publish(context.Background(), Event{DocumentID: "doc1", SenderID: "outro", Payload: p})
	}

	// O rápido recebe tudo: o lento não trava o hub.
	if got := received(fast); len(got) != 3 {
		t.Fatalf("cliente rápido: esperava 3 mensagens, recebeu %v", got)
	}
	// O lento recebeu só o que cabia no buffer e foi desconectado (canal fechado).
	if got := received(slow); len(got) != 1 {
		t.Errorf("cliente lento: esperava 1 mensagem antes de ser desconectado, recebeu %v", got)
	}
	if _, ok := <-slow.send; ok {
		t.Error("cliente lento deveria ter o canal fechado")
	}
}
