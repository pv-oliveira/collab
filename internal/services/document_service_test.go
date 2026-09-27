package services

import (
	"apis/internal/models"
	"apis/internal/repositories"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func TestDocumentServiceCreateStartsEmpty(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definido: pulando teste de integração")
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Documento precisa de um dono (FK); o CASCADE apaga o documento junto.
	user := &models.User{ID: uuid.NewString(), Email: uuid.NewString() + "@test.dev", Password: "hash", CreatedAt: time.Now()}
	if err := (&repositories.UserRepository{DB: db}).Create(user); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM users WHERE id = $1`, user.ID) })

	svc := &DocumentService{Repo: &repositories.DocumentRepository{DB: db}}
	doc, err := svc.Create(user.ID, "Novo")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Content != "" {
		t.Errorf("conteúdo inicial = %q, esperava vazio", doc.Content)
	}
}
