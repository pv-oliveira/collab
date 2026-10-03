package middleware

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	// Sem o Redis não dá para saber se o token foi revogado: quem chama
	// responde 503 em vez de aceitar um token possivelmente vazado.
	ErrRevocationUnavailable = errors.New("token revocation check unavailable")
)

// Claims são os dados do JWT que a API usa. ID é o jti: identifica este
// token específico, para o logout revogar só ele.
type Claims struct {
	UserID    string
	ID        string
	ExpiresAt time.Time
}

func GenerateToken(secret, userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"jti": uuid.NewString(),
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken valida assinatura e expiração e devolve as claims. Aceita
// apenas HS256, o mesmo algoritmo usado em GenerateToken. Não consulta a
// revogação: para isso, use Tokens.Parse.
func ParseToken(secret, tokenStr string) (*Claims, error) {
	var rc jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(tokenStr, &rc, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	// Sem jti (tokens emitidos antes da revogação existir) não há como revogar: recusa.
	if err != nil || rc.Subject == "" || rc.ID == "" {
		return nil, ErrInvalidToken
	}

	return &Claims{UserID: rc.Subject, ID: rc.ID, ExpiresAt: rc.ExpiresAt.Time}, nil
}

// Tokens é o ponto único de verificação (HTTP e WebSocket): assinatura +
// denylist de jti no Redis.
type Tokens struct {
	Secret string
	Redis  *redis.Client
}

func revokedKey(jti string) string { return "revoked:" + jti }

func (t *Tokens) Parse(ctx context.Context, tokenStr string) (*Claims, error) {
	claims, err := ParseToken(t.Secret, tokenStr)
	if err != nil {
		return nil, err
	}

	n, err := t.Redis.Exists(ctx, revokedKey(claims.ID)).Result()
	if err != nil {
		return nil, errors.Join(ErrRevocationUnavailable, err)
	}
	if n > 0 {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// Revoke põe o jti na denylist só pelo tempo que o token ainda valeria:
// depois disso ele já seria recusado pela expiração, e a chave some sozinha.
func (t *Tokens) Revoke(ctx context.Context, claims *Claims) error {
	ttl := time.Until(claims.ExpiresAt)
	if ttl <= 0 {
		return nil
	}
	return t.Redis.Set(ctx, revokedKey(claims.ID), 1, ttl).Err()
}
