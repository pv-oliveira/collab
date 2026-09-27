package db

import (
	"os"
	"testing"
)

func TestConnectFailsWhenDatabaseIsUnreachable(t *testing.T) {
	// Porta 1: nada escuta nela, a conexão é recusada na hora.
	db, err := Connect("postgres://u:p@127.0.0.1:1/x?sslmode=disable")
	if err == nil {
		_ = db.Close()
		t.Fatal("esperava erro com banco inacessível")
	}
}

func TestConnectSucceeds(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definido: pulando teste de integração")
	}

	db, err := Connect(url)
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
}
