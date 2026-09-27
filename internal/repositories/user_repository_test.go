package repositories

import (
	"apis/internal/models"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// testDB conecta no Postgres de teste (com as migrations aplicadas).
// Sem TEST_DATABASE_URL o teste é pulado, para `go test` rodar em qualquer máquina.
func testDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definido: pulando teste de integração")
	}

	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestUserRepositoryCreateDuplicateEmail(t *testing.T) {
	db := testDB(t)
	repo := &UserRepository{DB: db}

	email := uuid.NewString() + "@test.dev" // único por execução
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM users WHERE email = $1`, email) })

	newUser := func() *models.User {
		return &models.User{ID: uuid.NewString(), Email: email, Password: "hash", CreatedAt: time.Now()}
	}

	if err := repo.Create(newUser()); err != nil {
		t.Fatalf("primeiro cadastro falhou: %v", err)
	}

	err := repo.Create(newUser())
	if !errors.Is(err, models.ErrEmailTaken) {
		t.Fatalf("esperava ErrEmailTaken, recebeu %v", err)
	}
}
