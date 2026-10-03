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

func TestDocumentRepositoryUpdateContentKeepsTitle(t *testing.T) {
	db := testutil.DB(t)
	user := testutil.CreateUser(t, db)
	repo := &DocumentRepository{DB: db}
	created := time.Now().Add(-time.Hour).Truncate(time.Microsecond)
	doc := &models.Document{ID: uuid.NewString(), UserID: user.ID, Title: "Título", CreatedAt: created, UpdatedAt: created}
	if err := repo.Create(doc); err != nil {
		t.Fatal(err)
	}

	if err := repo.UpdateContent(doc.ID, "salvo sozinho", time.Now()); err != nil {
		t.Fatal(err)
	}

	got, err := repo.FindByIDAndUser(doc.ID, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "salvo sozinho" || got.Title != "Título" || !got.UpdatedAt.After(created) {
		t.Errorf("documento %+v: esperava conteúdo novo, mesmo título e updated_at avançado", got)
	}
}
