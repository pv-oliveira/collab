package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuthMiddlewareScheme(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokens, _ := newTokens(t)
	token, err := GenerateToken(tokens.Secret, "user-1")
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.GET("/", AuthMiddleware(tokens), func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString("userID"))
	})

	tests := []struct {
		name   string
		header string
		want   int
	}{
		{"Bearer", "Bearer " + token, http.StatusOK},
		{"bearer minúsculo", "bearer " + token, http.StatusOK},
		{"BEARER maiúsculo", "BEARER " + token, http.StatusOK},
		{"outro esquema", "Foo " + token, http.StatusUnauthorized},
		{"Basic", "Basic " + token, http.StatusUnauthorized},
		{"só o token", token, http.StatusUnauthorized},
		{"Bearer sem token", "Bearer ", http.StatusUnauthorized},
		{"sem header", "", http.StatusUnauthorized},
		{"token inválido", "Bearer abc.def.ghi", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.want {
				t.Errorf("header %q: status %d, esperava %d", tt.header, w.Code, tt.want)
			}
		})
	}
}

func TestAuthMiddlewareRevocation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokens, mr := newTokens(t)
	r := gin.New()
	r.GET("/", AuthMiddleware(tokens), func(c *gin.Context) { c.Status(http.StatusOK) })
	get := func(token string) int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	revoked, _ := GenerateToken(tokens.Secret, "user-1")
	claims, _ := ParseToken(tokens.Secret, revoked)
	if err := tokens.Revoke(context.Background(), claims); err != nil {
		t.Fatal(err)
	}
	if got := get(revoked); got != http.StatusUnauthorized {
		t.Errorf("token revogado: status %d, esperava 401", got)
	}

	valid, _ := GenerateToken(tokens.Secret, "user-1")
	mr.Close()
	if got := get(valid); got != http.StatusServiceUnavailable {
		t.Errorf("Redis fora: status %d, esperava 503", got)
	}
}
