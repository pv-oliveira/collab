package repositories

import (
	"apis/internal/models"
	"database/sql"
	"fmt"
)

type DocumentRepository struct {
	DB *sql.DB
}

func (r *DocumentRepository) Create(document *models.Document) error {
	query := `INSERT INTO documents (id, title, content, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`

	_, err := r.DB.Exec(query, document.ID, document.Title, document.Content, document.CreatedAt, document.UpdatedAt)
	if err != nil {
		fmt.Println("Error inserting document:", err.Error())
	}
	return nil
}

func (r *DocumentRepository) FindByID(id string) (*models.Document, error) {
	query := `SELECT id, title, content, created_at, updated_at FROM documents WHERE id = $1`

	var document models.Document
	err := r.DB.QueryRow(query, id).Scan(&document.ID, &document.Title, &document.Content, &document.CreatedAt, &document.UpdatedAt)
	return &document, err
}
