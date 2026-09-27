package services

import (
	"apis/internal/repositories"
	"apis/internal/testutil"
	"testing"
)

func TestDocumentServiceCreateStartsEmpty(t *testing.T) {
	db := testutil.DB(t)
	user := testutil.CreateUser(t, db)

	svc := &DocumentService{Repo: &repositories.DocumentRepository{DB: db}}
	doc, err := svc.Create(user.ID, "Novo")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Content != "" {
		t.Errorf("conteúdo inicial = %q, esperava vazio", doc.Content)
	}
}

func TestDocumentServiceUpdateRespectsOwner(t *testing.T) {
	db := testutil.DB(t)
	owner := testutil.CreateUser(t, db)
	intruder := testutil.CreateUser(t, db)
	svc := &DocumentService{Repo: &repositories.DocumentRepository{DB: db}}

	doc, err := svc.Create(owner.ID, "Original")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Update(owner.ID, doc.ID, "Do dono", "ok"); err != nil {
		t.Fatalf("dono não conseguiu editar: %v", err)
	}

	if _, err := svc.Update(intruder.ID, doc.ID, "Invadido", "hack"); err == nil {
		t.Fatal("outro usuário conseguiu editar o documento")
	}

	// Além do erro, garante que nada foi gravado.
	got, err := svc.GetByID(owner.ID, doc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Do dono" || got.Content != "ok" {
		t.Errorf("documento alterado por outro usuário: title=%q content=%q", got.Title, got.Content)
	}

	if _, err := svc.GetByID(intruder.ID, doc.ID); err == nil {
		t.Error("outro usuário conseguiu ler o documento")
	}
}
