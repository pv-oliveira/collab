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
	query := `INSERT INTO documents (id, user_id, title, content, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.DB.Exec(query, document.ID, document.UserID, document.Title, document.Content, document.CreatedAt, document.UpdatedAt)
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

func (r *DocumentRepository) Update(doc *models.Document) error {
	query := `UPDATE documents SET title=$1, content=$2, updated_at=$3 WHERE id=$4 AND user_id=$5`
	_, err := r.DB.Exec(
		query,
		doc.Title, doc.Content, doc.UpdatedAt, doc.ID, doc.UserID,
	)
	if err != nil {
		fmt.Println("Error updating document:", err.Error())
	}
	return err
}

func (r *DocumentRepository) FindByIDAndUser(id, userID string) (*models.Document, error) {
	query := `SELECT id, user_id, title, content, created_at, updated_at
         FROM documents WHERE id=$1 AND user_id=$2`
	row := r.DB.QueryRow(
		query,
		id, userID,
	)

	var doc models.Document
	err := row.Scan(&doc.ID, &doc.UserID, &doc.Title, &doc.Content, &doc.CreatedAt, &doc.UpdatedAt)
	return &doc, err
}

func (r *DocumentRepository) FindByUser(userID string) ([]*models.Document, error) {
	var documents []*models.Document
	query := `SELECT id, user_id, title, content, created_at, updated_at
         FROM documents WHERE user_id=$1`
	rows, err := r.DB.Query(query, userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var doc models.Document
		err = rows.Scan(&doc.ID, &doc.UserID, &doc.Title, &doc.Content, &doc.CreatedAt, &doc.UpdatedAt)
		if err != nil {
			return nil, err
		}
		documents = append(documents, &doc)
	}

	return documents, nil
}
