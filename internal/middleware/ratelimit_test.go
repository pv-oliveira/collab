package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// rateLimitRouter monta duas rotas limitadas a 2 requisições por minuto, com
// SetTrustedProxies(nil) como no main.go.
func rateLimitRouter(t *testing.T) (*gin.Engine, *miniredis.Miniredis) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	mr := miniredis.RunT(t)
	limit := RateLimit(redis.NewClient(&redis.Options{Addr: mr.Addr()}), 2, time.Minute)

	r := gin.New()
	if err := r.SetTrustedProxies(nil); err != nil {
		t.Fatal(err)
	}
	r.POST("/auth/login", limit, func(c *gin.Context) { c.Status(http.StatusOK) })
	r.POST("/auth/register", limit, func(c *gin.Context) { c.Status(http.StatusOK) })
	return r, mr
}

func post(r *gin.Engine, path, ip, forwardedFor string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, nil)
	req.RemoteAddr = ip + ":12345"
	if forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRateLimitBlocksAfterLimit(t *testing.T) {
	r, _ := rateLimitRouter(t)

	for i := range 2 {
		if w := post(r, "/auth/login", "1.1.1.1", ""); w.Code != http.StatusOK {
			t.Fatalf("requisição %d: status %d, esperava 200", i+1, w.Code)
		}
	}

	w := post(r, "/auth/login", "1.1.1.1", "")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status %d, esperava 429", w.Code)
	}
	if got := w.Header().Get("Retry-After"); got != "60" {
		t.Errorf("Retry-After %q, esperava \"60\"", got)
	}
}

func TestRateLimitReleasesAfterWindow(t *testing.T) {
	r, mr := rateLimitRouter(t)
	for range 3 {
		post(r, "/auth/login", "1.1.1.1", "")
	}

	mr.FastForward(time.Minute)

	if w := post(r, "/auth/login", "1.1.1.1", ""); w.Code != http.StatusOK {
		t.Fatalf("status %d, esperava 200 após a janela", w.Code)
	}
}

func TestRateLimitCountsPerIPAndRoute(t *testing.T) {
	r, _ := rateLimitRouter(t)
	for range 3 {
		post(r, "/auth/login", "1.1.1.1", "")
	}

	if w := post(r, "/auth/login", "2.2.2.2", ""); w.Code != http.StatusOK {
		t.Errorf("outro IP: status %d, esperava 200", w.Code)
	}
	if w := post(r, "/auth/register", "1.1.1.1", ""); w.Code != http.StatusOK {
		t.Errorf("outra rota: status %d, esperava 200", w.Code)
	}
}

func TestRateLimitIgnoresSpoofedForwardedFor(t *testing.T) {
	// Um IP falso diferente a cada requisição não pode zerar o contador.
	r, _ := rateLimitRouter(t)
	for _, fake := range []string{"9.9.9.1", "9.9.9.2"} {
		post(r, "/auth/login", "1.1.1.1", fake)
	}

	if w := post(r, "/auth/login", "1.1.1.1", "9.9.9.3"); w.Code != http.StatusTooManyRequests {
		t.Fatalf("status %d, esperava 429", w.Code)
	}
}

func TestRateLimitFailsOpenWhenRedisIsDown(t *testing.T) {
	r, mr := rateLimitRouter(t)
	mr.Close()

	if w := post(r, "/auth/login", "1.1.1.1", ""); w.Code != http.StatusOK {
		t.Fatalf("status %d, esperava 200 com o Redis fora", w.Code)
	}
}
