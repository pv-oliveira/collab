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

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
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

	r := gin.Default()

	authRepo := &repositories.UserRepository{DB: database}
	authService := &services.AuthService{
		Repo:      authRepo,
		JWTSecret: cfg.JWTSecret,
	}
	authHandler := &handlers.AuthHandler{Service: authService}

	r.POST("/auth/register", authHandler.Register)
	r.POST("/auth/login", authHandler.Login)

	// Rotas protegidas
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))

	protected.POST("/documents", handler.Create)
	protected.GET("/documents/:id", handler.Get)
	protected.GET("/documents", handler.List)
	protected.PUT("/documents/:id", handler.Update)

	// WebSocket: fora do grupo protegido porque o token vem no subprotocolo,
	// e não no header Authorization (o próprio handler valida).
	if cfg.RedisURL == "" {
		log.Fatal("REDIS_URL is required")
	}
	ctx := context.Background()
	bus, err := ws.NewRedisBus(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatal("redis: ", err)
	}
	hub := ws.NewHub(bus)
	go func() {
		log.Fatal("ws hub: ", hub.Run(ctx))
	}()
	r.GET("/documents/:id/ws", ws.NewWSHandler(hub, service, cfg.JWTSecret, cfg.AllowedOrigins).Handle)

	log.Fatal(r.Run(":8080"))
}
