package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func corsRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS([]string{"http://localhost:5173"}))
	protected := r.Group("/")
	protected.Use(AuthMiddleware("secret"))
	protected.GET("/documents", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func request(r *gin.Engine, method, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/documents", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if method == http.MethodOptions {
		req.Header.Set("Access-Control-Request-Method", "GET")
		req.Header.Set("Access-Control-Request-Headers", "authorization")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCORSPreflightFromAllowedOrigin(t *testing.T) {
	// Preflight não leva token: precisa ser respondido antes da autenticação.
	w := request(corsRouter(), http.MethodOptions, "http://localhost:5173")

	if w.Code != http.StatusNoContent {
		t.Fatalf("status %d, esperava 204", w.Code)
	}
	want := map[string]string{
		"Access-Control-Allow-Origin":  "http://localhost:5173",
		"Access-Control-Allow-Methods": "GET, POST, PUT, OPTIONS",
		"Access-Control-Allow-Headers": "Authorization, Content-Type",
		"Access-Control-Max-Age":       "600",
		"Vary":                         "Origin",
	}
	for k, v := range want {
		if got := w.Header().Get(k); got != v {
			t.Errorf("%s = %q, esperava %q", k, got, v)
		}
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("Allow-Credentials não deveria ser enviado, veio %q", got)
	}
}

func TestCORSDisallowedOrigin(t *testing.T) {
	for _, method := range []string{http.MethodOptions, http.MethodGet} {
		w := request(corsRouter(), method, "https://evil.com")
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("%s de origem não permitida recebeu Allow-Origin %q", method, got)
		}
	}
}

func TestCORSRequestWithoutOriginIsUnaffected(t *testing.T) {
	// curl e servidores não mandam Origin: sem headers CORS, e a auth continua valendo.
	w := request(corsRouter(), http.MethodGet, "")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status %d, esperava 401 (sem token)", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("sem Origin não deveria ter Allow-Origin, veio %q", got)
	}
}

func TestCORSActualRequestFromAllowedOrigin(t *testing.T) {
	// A requisição real (não-preflight) também precisa do Allow-Origin, senão o
	// navegador descarta a resposta. Aqui ela segue para a auth (401 sem token).
	w := request(corsRouter(), http.MethodGet, "http://localhost:5173")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("Allow-Origin = %q na requisição real", got)
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status %d, esperava 401: CORS não substitui autenticação", w.Code)
	}
}

func TestOriginAllowed(t *testing.T) {
	allowed := []string{"http://localhost:5173"}
	if !OriginAllowed("http://LOCALHOST:5173", allowed) {
		t.Error("comparação deveria ignorar maiúsculas")
	}
	if OriginAllowed("http://localhost:3000", allowed) || OriginAllowed("", allowed) {
		t.Error("origem fora da lista (ou vazia) não pode ser permitida")
	}
}
