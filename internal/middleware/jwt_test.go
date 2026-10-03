package middleware

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
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
		{"alg=none", sign(jwt.SigningMethodNone, jwt.MapClaims{"sub": "x", "jti": "j", "exp": future}, jwt.UnsafeAllowNoneSignatureType), "", true},
		{"HS384 (outro algoritmo)", sign(jwt.SigningMethodHS384, jwt.MapClaims{"sub": "x", "jti": "j", "exp": future}, []byte(secret)), "", true},
		{"outro segredo", sign(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "x", "jti": "j", "exp": future}, []byte("outro")), "", true},
		{"sem sub", sign(jwt.SigningMethodHS256, jwt.MapClaims{"jti": "j", "exp": future}, []byte(secret)), "", true},
		{"sub não-string", sign(jwt.SigningMethodHS256, jwt.MapClaims{"sub": 123, "jti": "j", "exp": future}, []byte(secret)), "", true},
		{"sem jti (token anterior à revogação)", sign(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "x", "exp": future}, []byte(secret)), "", true},
		{"sem exp", sign(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "x", "jti": "j"}, []byte(secret)), "", true},
		{"expirado", sign(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "x", "jti": "j", "exp": time.Now().Add(-time.Minute).Unix()}, []byte(secret)), "", true},
		{"lixo", "nao-e-um-jwt", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ParseToken(secret, tt.token)
			if (err != nil) != tt.wantErr {
				t.Fatalf("erro = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && claims.UserID != tt.wantID {
				t.Errorf("userID = %q, want %q", claims.UserID, tt.wantID)
			}
		})
	}
}

func TestGenerateTokenUniqueJTI(t *testing.T) {
	a, _ := GenerateToken("secret", "user-1")
	b, _ := GenerateToken("secret", "user-1")
	ca, err := ParseToken("secret", a)
	if err != nil {
		t.Fatal(err)
	}
	cb, err := ParseToken("secret", b)
	if err != nil {
		t.Fatal(err)
	}
	if ca.ID == "" || ca.ID == cb.ID {
		t.Errorf("jti precisa ser único por token: %q e %q", ca.ID, cb.ID)
	}
}

func newTokens(t *testing.T) (*Tokens, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	return &Tokens{Secret: "secret", Redis: redis.NewClient(&redis.Options{Addr: mr.Addr()})}, mr
}

func TestTokensRevoke(t *testing.T) {
	ctx := context.Background()
	tokens, mr := newTokens(t)
	token, _ := GenerateToken(tokens.Secret, "user-1")
	other, _ := GenerateToken(tokens.Secret, "user-1")

	claims, err := tokens.Parse(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if err := tokens.Revoke(ctx, claims); err != nil {
		t.Fatal(err)
	}

	if _, err := tokens.Parse(ctx, token); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("token revogado: erro %v, esperava ErrInvalidToken", err)
	}
	if _, err := tokens.Parse(ctx, other); err != nil {
		t.Errorf("outro token do mesmo usuário deveria continuar válido: %v", err)
	}

	// A chave vive só enquanto o token viveria.
	ttl := mr.TTL("revoked:" + claims.ID)
	if ttl <= 23*time.Hour || ttl > 24*time.Hour {
		t.Errorf("TTL da revogação = %v, esperava ~24h", ttl)
	}
	mr.FastForward(24 * time.Hour)
	if mr.Exists("revoked:" + claims.ID) {
		t.Error("a revogação deveria expirar junto com o token")
	}
}

func TestTokensFailClosedWhenRedisIsDown(t *testing.T) {
	tokens, mr := newTokens(t)
	token, _ := GenerateToken(tokens.Secret, "user-1")
	mr.Close()

	if _, err := tokens.Parse(context.Background(), token); !errors.Is(err, ErrRevocationUnavailable) {
		t.Errorf("erro %v, esperava ErrRevocationUnavailable", err)
	}
}
