package repositories

import (
	"apis/internal/models"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
)

// Código do Postgres para violação de UNIQUE (aqui, users.email).
const uniqueViolation = "23505"

type UserRepository struct {
	DB *sql.DB
}

func (r *UserRepository) Create(user *models.User) error {
	query := `INSERT INTO users (id, email, password, created_at)
         VALUES ($1, $2, $3, $4)`
	_, err := r.DB.Exec(
		query,
		user.ID, user.Email, user.Password, user.CreatedAt,
	)
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == uniqueViolation {
		return models.ErrEmailTaken
	}
	if err != nil {
		fmt.Println("Error inserting user:", err.Error())
	}

	return err
}

func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	query := `SELECT id, email, password, created_at FROM users WHERE email=$1`
	row := r.DB.QueryRow(query, email)

	var user models.User
	err := row.Scan(&user.ID, &user.Email, &user.Password, &user.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
