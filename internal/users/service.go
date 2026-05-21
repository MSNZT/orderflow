package users

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Service struct {
	repo   UserRepository
	hasher PasswordHasher
}

var (
	ErrInvalidEmail    = errors.New("invalid email")
	ErrInvalidPassword = errors.New("invalid password")
)

func NewService(repo UserRepository, hasher PasswordHasher) *Service {
	return &Service{
		repo:   repo,
		hasher: hasher,
	}
}

func (s *Service) Register(ctx context.Context, email, password string) (*User, error) {
	const op = "users.service.Register"

	email = strings.TrimSpace(strings.ToLower(email))

	if email == "" || !strings.Contains(email, "@") {
		return nil, fmt.Errorf("%s: %w", op, ErrInvalidEmail)
	}

	if len(password) < 8 {
		return nil, fmt.Errorf("%s: %w", op, ErrInvalidPassword)
	}

	passwordHash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("%s: hash password: %w", op, err)
	}

	user := User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
		Role:         RoleCustomer,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &user, nil
}
