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

func (s *DocumentService) Create(userID, title string) (*models.Document, error) {
	document := &models.Document{
		ID:        uuid.New().String(),
		UserID:    userID,
		Title:     title,
		Content:   "TESTE",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return document, s.Repo.Create(document)
}

func (s *DocumentService) GetByID(userID, docID string) (*models.Document, error) {
	return s.Repo.FindByIDAndUser(docID, userID)
}

func (s *DocumentService) ListByUser(userID string) ([]*models.Document, error) {
	return s.Repo.FindByUser(userID)
}

func (s *DocumentService) Update(userID, docID, title, content string) (*models.Document, error) {
	doc, err := s.Repo.FindByIDAndUser(docID, userID)
	if err != nil {
		return nil, err
	}

	doc.Title = title
	doc.Content = content
	doc.UpdatedAt = time.Now()

	return doc, s.Repo.Update(doc)
}
