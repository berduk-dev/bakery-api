package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/berduk-dev/bad-da-yo/internal/repo"
)

type Service struct {
	repo repo.Repository
}

func New(repo repo.Repository) Service {
	return Service{
		repo: repo,
	}
}
func (s *Service) CreatePrize(ctx context.Context, prizeName string) (string, error) {
	code, err := generateCode(6)

	err = s.repo.CreatePrize(ctx, prizeName, code)
	if err != nil {
		return "", fmt.Errorf("error repo.CreatePrize: %w", err)
	}

	return code, nil
}

// Генерация случайного промокода (буквы + цифры)
func generateCode(length int) (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	code := make([]byte, length)
	randomBytes := make([]byte, length)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	for i := range code {
		code[i] = charset[int(randomBytes[i])%len(charset)]
	}

	return string(code), nil
}
