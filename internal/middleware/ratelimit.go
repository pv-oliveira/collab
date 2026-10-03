package middleware

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimit permite até limit requisições por janela, por rota e IP. O contador
// fica no Redis para valer entre todas as réplicas (janela fixa: na virada
// passam até 2× o limite, aceitável contra brute-force).
// O IP vem de c.ClientIP(): o router precisa de SetTrustedProxies(nil) para
// que um X-Forwarded-For falsificado não troque o IP a cada requisição.
func RateLimit(rdb *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "ratelimit:" + c.FullPath() + ":" + c.ClientIP()

		// MULTI: criar com TTL e incrementar juntos. Com INCR e EXPIRE separados,
		// uma queda entre os dois deixaria a chave sem TTL (bloqueio eterno).
		var count *redis.IntCmd
		var ttl *redis.DurationCmd
		_, err := rdb.TxPipelined(c, func(p redis.Pipeliner) error {
			p.SetNX(c, key, 0, window)
			count = p.Incr(c, key)
			ttl = p.TTL(c, key)
			return nil
		})
		if err != nil {
			// Fail open: sem Redis o login continua funcionando, só sem limite.
			log.Println("ratelimit:", err)
			c.Next()
			return
		}

		if count.Val() > int64(limit) {
			retry := max(int(ttl.Val().Seconds()), 1)
			c.Header("Retry-After", strconv.Itoa(retry))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		c.Next()
	}
}
