package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuthMiddlewareScheme(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const secret = "secret"
	token, err := GenerateToken(secret, "user-1")
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.GET("/", AuthMiddleware(secret), func(c *gin.Context) {
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
