package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// OriginAllowed diz se a origem está na lista (sem diferenciar maiúsculas).
// É a mesma regra para CORS e para o WebSocket.
func OriginAllowed(origin string, allowed []string) bool {
	for _, a := range allowed {
		if origin != "" && strings.EqualFold(origin, a) {
			return true
		}
	}
	return false
}

// CORS autoriza navegadores das origens permitidas a chamar a API. Precisa
// rodar antes da autenticação: o preflight (OPTIONS) não leva o token.
// Sem Allow-Credentials, pois a autenticação é por Bearer, não por cookie.
func CORS(allowed []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		// A resposta muda conforme a origem: caches não podem reaproveitá-la entre origens.
		h.Add("Vary", "Origin")

		if origin := c.GetHeader("Origin"); OriginAllowed(origin, allowed) {
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			h.Set("Access-Control-Max-Age", "600")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
