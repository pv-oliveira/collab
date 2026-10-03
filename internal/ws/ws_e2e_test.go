package ws

import (
	"apis/internal/middleware"
	"apis/internal/repositories"
	"apis/internal/services"
	"apis/internal/testutil"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	e2eSecret = "ws-e2e-secret"
	e2eOrigin = "http://localhost:5173"
)

// Teste ponta a ponta: conexões WebSocket reais passando por Handle,
// ReadPump e WritePump, com Postgres de teste e Redis em memória.
func TestWebSocketEndToEnd(t *testing.T) {
	db := testutil.DB(t)
	owner := testutil.CreateUser(t, db)
	intruder := testutil.CreateUser(t, db)

	docs := &services.DocumentService{Repo: &repositories.DocumentRepository{DB: db}}
	doc, err := docs.Create(owner.ID, "Compartilhado")
	if err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	tokens := testutil.Tokens(t, e2eSecret)
	r.GET("/documents/:id/ws", NewWSHandler(startHubs(t, 1)[0], docs, tokens, []string{e2eOrigin}).Handle)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/documents/" + doc.ID + "/ws"

	ownerToken, _ := middleware.GenerateToken(e2eSecret, owner.ID)
	intruderToken, _ := middleware.GenerateToken(e2eSecret, intruder.ID)
	revokedToken, _ := middleware.GenerateToken(e2eSecret, owner.ID)
	revoked, _ := middleware.ParseToken(e2eSecret, revokedToken)
	if err := tokens.Revoke(context.Background(), revoked); err != nil {
		t.Fatal(err)
	}

	dial := func(token, origin string) (*websocket.Conn, *http.Response, error) {
		d := websocket.Dialer{Subprotocols: []string{tokenProtocol, token}}
		h := http.Header{}
		if origin != "" {
			h.Set("Origin", origin)
		}
		return d.Dial(url, h)
	}

	t.Run("recusas antes do upgrade", func(t *testing.T) {
		cases := []struct {
			name, token, origin string
			want                int
		}{
			{"sem token", "", e2eOrigin, http.StatusUnauthorized},
			{"token revogado (logout)", revokedToken, e2eOrigin, http.StatusUnauthorized},
			{"documento de outro usuário", intruderToken, e2eOrigin, http.StatusNotFound},
			{"origem não permitida", ownerToken, "https://evil.com", http.StatusForbidden},
		}
		for _, c := range cases {
			conn, resp, err := dial(c.token, c.origin)
			if err == nil {
				_ = conn.Close()
				t.Errorf("%s: conexão aceita, esperava %d", c.name, c.want)
				continue
			}
			if resp == nil || resp.StatusCode != c.want {
				t.Errorf("%s: resposta %v, esperava %d", c.name, resp, c.want)
			}
		}
	})

	t.Run("edição chega na outra aba", func(t *testing.T) {
		tab1, resp, err := dial(ownerToken, e2eOrigin)
		if err != nil {
			t.Fatalf("aba 1: %v", err)
		}
		defer func() { _ = tab1.Close() }()
		// O navegador fecha a conexão se o servidor não devolver o subprotocolo.
		if got := resp.Header.Get("Sec-WebSocket-Protocol"); got != tokenProtocol {
			t.Errorf("subprotocolo negociado = %q, esperava %q", got, tokenProtocol)
		}

		tab2, _, err := dial(ownerToken, e2eOrigin)
		if err != nil {
			t.Fatalf("aba 2: %v", err)
		}
		defer func() { _ = tab2.Close() }()

		// Dá tempo do hub registrar as duas conexões.
		time.Sleep(100 * time.Millisecond)

		if err := tab1.WriteMessage(websocket.TextMessage, []byte("olá")); err != nil {
			t.Fatal(err)
		}

		_ = tab2.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := tab2.ReadMessage()
		if err != nil || string(msg) != "olá" {
			t.Fatalf("aba 2: mensagem %q, erro %v", msg, err)
		}

		// A aba que enviou não recebe eco.
		_ = tab1.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
		if _, msg, err := tab1.ReadMessage(); err == nil {
			t.Errorf("aba 1 recebeu o próprio eco: %q", msg)
		}
	})
}
