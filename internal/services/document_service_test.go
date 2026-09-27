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
