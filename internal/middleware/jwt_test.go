package middleware

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestParseToken(t *testing.T) {
	const secret = "secret"
	sign := func(method jwt.SigningMethod, claims jwt.MapClaims, key any) string {
		t.Helper()
		s, err := jwt.NewWithClaims(method, claims).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	future := time.Now().Add(time.Hour).Unix()

	valid, err := GenerateToken(secret, "user-1")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		token   string
		wantID  string
		wantErr bool
	}{
		{"válido", valid, "user-1", false},
		{"alg=none", sign(jwt.SigningMethodNone, jwt.MapClaims{"sub": "x", "exp": future}, jwt.UnsafeAllowNoneSignatureType), "", true},
		{"HS384 (outro algoritmo)", sign(jwt.SigningMethodHS384, jwt.MapClaims{"sub": "x", "exp": future}, []byte(secret)), "", true},
		{"outro segredo", sign(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "x", "exp": future}, []byte("outro")), "", true},
		{"sem sub", sign(jwt.SigningMethodHS256, jwt.MapClaims{"exp": future}, []byte(secret)), "", true},
		{"sub não-string", sign(jwt.SigningMethodHS256, jwt.MapClaims{"sub": 123, "exp": future}, []byte(secret)), "", true},
		{"expirado", sign(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "x", "exp": time.Now().Add(-time.Minute).Unix()}, []byte(secret)), "", true},
		{"lixo", "nao-e-um-jwt", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := ParseToken(secret, tt.token)
			if (err != nil) != tt.wantErr {
				t.Fatalf("erro = %v, wantErr %v", err, tt.wantErr)
			}
			if id != tt.wantID {
				t.Errorf("userID = %q, want %q", id, tt.wantID)
			}
		})
	}
}
