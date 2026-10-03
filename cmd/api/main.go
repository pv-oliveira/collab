package main

import (
	"apis/internal/config"
	"apis/internal/db"
	"apis/internal/handlers"
	"apis/internal/middleware"
	"apis/internal/repositories"
	"apis/internal/services"
	"apis/internal/ws"
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	// .env é opcional: em Docker/produção as variáveis vêm do ambiente.
	_ = godotenv.Load()
	cfg := config.Load()

	database, err := db.Connect(cfg.DBUrl)
	if err != nil {
		log.Fatal(err)
	}

	repo := &repositories.DocumentRepository{DB: database}
	service := &services.DocumentService{Repo: repo}
	handler := &handlers.DocumentHandler{Service: service}

	if cfg.RedisURL == "" {
		log.Fatal("REDIS_URL is required")
	}
	ctx := context.Background()
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatal("redis: ", err)
	}
	rdb := redis.NewClient(redisOpts)
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("redis: ", err)
	}

	r := gin.Default()
	// Sem proxy na frente: o IP vem da conexão TCP. Confiar no X-Forwarded-For
	// deixaria o cliente escolher o IP e escapar do rate limit.
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatal(err)
	}
	// Antes de qualquer rota: o preflight (OPTIONS) chega sem token.
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	authRepo := &repositories.UserRepository{DB: database}
	authService := &services.AuthService{
		Repo:      authRepo,
		JWTSecret: cfg.JWTSecret,
	}
	tokens := &middleware.Tokens{Secret: cfg.JWTSecret, Redis: rdb}
	authHandler := &handlers.AuthHandler{Service: authService, Tokens: tokens}

	// Contra brute-force de senha e criação de contas em massa.
	authLimit := middleware.RateLimit(rdb, 10, time.Minute)
	r.POST("/auth/register", authLimit, authHandler.Register)
	r.POST("/auth/login", authLimit, authHandler.Login)

	// Rotas protegidas
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware(tokens))

	protected.POST("/auth/logout", authHandler.Logout)
	protected.POST("/documents", handler.Create)
	protected.GET("/documents/:id", handler.Get)
	protected.GET("/documents", handler.List)
	protected.PUT("/documents/:id", handler.Update)

	// WebSocket: fora do grupo protegido porque o token vem no subprotocolo,
	// e não no header Authorization (o próprio handler valida).
	hub := ws.NewHub(ws.NewRedisBus(rdb))
	go func() {
		log.Fatal("ws hub: ", hub.Run(ctx))
	}()
	r.GET("/documents/:id/ws", ws.NewWSHandler(hub, service, tokens, cfg.AllowedOrigins).Handle)

	log.Fatal(r.Run(":8080"))
}
