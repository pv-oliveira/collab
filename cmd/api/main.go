package main

import (
	"apis/internal/config"
	"apis/internal/db"
	"apis/internal/handlers"
	"apis/internal/middleware"
	"apis/internal/repositories"
	"apis/internal/services"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
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
	protected.PUT("/documents/:id", handler.Update)

	r.Run(":8080")
}
