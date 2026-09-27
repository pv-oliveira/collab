package middleware

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateToken(secret, userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken valida o JWT e devolve o userID (claim "sub").
// Aceita apenas HS256, o mesmo algoritmo usado em GenerateToken.
func ParseToken(secret, tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}

	userID, err := token.Claims.GetSubject()
	if err != nil || userID == "" {
		return "", errors.New("invalid token")
	}

	return userID, nil
}
