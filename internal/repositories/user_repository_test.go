package repositories

import (
	"apis/internal/models"
	"apis/internal/testutil"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUserRepositoryCreateDuplicateEmail(t *testing.T) {
	db := testutil.DB(t)
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
