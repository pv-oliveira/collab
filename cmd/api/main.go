package main

import (
	"apis/internal/config"
	"apis/internal/db"
	"apis/internal/handlers"
	"apis/internal/repositories"
	"apis/internal/services"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	database, err := db.Connect(cfg.DBUrl)
	if err != nil {
		log.Fatal(err)
	}

	repo := &repositories.DocumentRepository{DB: database}
	service := &services.DocumentService{Repo: repo}
	handler := &handlers.DocumentHandler{Service: service}

	r := gin.Default()

	r.POST("/documents", handler.Create)

	r.Run(":8080")
}
