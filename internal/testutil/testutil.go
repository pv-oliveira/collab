// Package testutil reúne helpers dos testes de integração. Só é importado
// por arquivos _test.go, então não entra no binário da API.
package testutil

import (
	"apis/internal/middleware"
	"apis/internal/models"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// DB conecta no Postgres de teste (com as migrations aplicadas). Sem
// TEST_DATABASE_URL o teste é pulado, para `go test` rodar em qualquer máquina.
func DB(t testing.TB) *sql.DB {
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

// CreateUser insere um usuário com e-mail único e o remove ao fim do teste
// (o ON DELETE CASCADE leva os documentos dele junto). Usa SQL direto para
// não importar repositories, cujos próprios testes importam este pacote.
func CreateUser(t testing.TB, db *sql.DB) *models.User {
	t.Helper()
	user := &models.User{
		ID:        uuid.NewString(),
		Email:     uuid.NewString() + "@test.dev",
		Password:  "hash",
		CreatedAt: time.Now(),
	}
	_, err := db.Exec(`INSERT INTO users (id, email, password, created_at) VALUES ($1, $2, $3, $4)`,
		user.ID, user.Email, user.Password, user.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM users WHERE id = $1`, user.ID) })
	return user
}

// Tokens devolve a verificação de JWT ligada a um Redis em memória.
func Tokens(t testing.TB, secret string) *middleware.Tokens {
	t.Helper()
	mr := miniredis.RunT(t)
	return &middleware.Tokens{Secret: secret, Redis: redis.NewClient(&redis.Options{Addr: mr.Addr()})}
}
