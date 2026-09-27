package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUserJSONNeverExposesPassword(t *testing.T) {
	user := User{ID: "u1", Email: "a@a.com", Password: "$2a$10$hash"}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatal(err)
	}

	got := string(data)
	if strings.Contains(got, "hash") || strings.Contains(strings.ToLower(got), "password") {
		t.Fatalf("senha exposta no JSON: %s", got)
	}
	for _, key := range []string{`"id"`, `"email"`, `"created_at"`} {
		if !strings.Contains(got, key) {
			t.Errorf("chave %s ausente em %s", key, got)
		}
	}
}
