package middleware

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware exige um Bearer válido e não revogado. Deixa no contexto
// "userID" e "claims" (usadas pelo logout).
func AuthMiddleware(tokens *Tokens) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			c.Abort()
			return
		}

		// Esquema "Bearer" sem diferenciar maiúsculas (RFC 6750).
		scheme, token, ok := strings.Cut(authHeader, " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token format"})
			c.Abort()
			return
		}

		claims, err := tokens.Parse(c, token)
		if errors.Is(err, ErrRevocationUnavailable) {
			log.Println("auth:", err)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "service unavailable"})
			return
		}
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("claims", claims)
		c.Next()
	}
}
