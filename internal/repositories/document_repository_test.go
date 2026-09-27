package repositories

import (
	"apis/internal/models"
	"apis/internal/testutil"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDocumentRepositoryCreateReturnsInsertError(t *testing.T) {
	db := testutil.DB(t)
	repo := &DocumentRepository{DB: db}

	// Dono inexistente: a FK recusa o INSERT, e o erro precisa chegar a quem chamou.
	doc := &models.Document{ID: uuid.NewString(), UserID: uuid.NewString(), CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.Create(doc); err == nil {
		t.Fatal("Create retornou nil para um INSERT que falhou")
	}
}

func TestDocumentRepositoryFindByUserEmptyIsNotNil(t *testing.T) {
	db := testutil.DB(t)
	user := testutil.CreateUser(t, db)

	docs, err := (&DocumentRepository{DB: db}).FindByUser(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	// nil vira `null` no JSON; slice vazia vira `[]`.
	if docs == nil {
		t.Error("FindByUser retornou nil para usuário sem documentos")
	}
}
