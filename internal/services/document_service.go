package services

import (
	"apis/internal/models"
	"apis/internal/repositories"
	"time"

	"github.com/google/uuid"
)

type DocumentService struct {
	Repo *repositories.DocumentRepository
}

func (s *DocumentService) Create(title string) (*models.Document, error) {
	document := &models.Document{
		ID:        uuid.New().String(),
		Title:     title,
		Content:   "TESTE",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return document, s.Repo.Create(document)
}
