package services

import (
	"apis/internal/middleware"
	"apis/internal/models"
	"apis/internal/repositories"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	Repo      *repositories.UserRepository
	JWTSecret string
}

func (s *AuthService) Register(email, password string) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		ID:        uuid.New().String(),
		Email:     email,
		Password:  string(hash),
		CreatedAt: time.Now(),
	}

	return user, s.Repo.Create(user)
}

func (s *AuthService) Login(email, password string) (string, error) {
	user, err := s.Repo.FindByEmail(email)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	token, err := middleware.GenerateToken(s.JWTSecret, user.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}
