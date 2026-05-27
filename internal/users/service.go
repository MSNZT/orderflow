package users

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
}

type Service struct {
	repo   UserRepository
	hasher PasswordHasher
}

var (
	ErrInvalidEmail       = errors.New("invalid email")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

func NewService(repo UserRepository, hasher PasswordHasher) *Service {
	return &Service{
		repo:   repo,
		hasher: hasher,
	}
}

func (s *Service) Register(ctx context.Context, email, password string) (*User, error) {
	const op = "users.service.Register"

	email = normalizeEmail(email)
	if !isEmailValid(email) {
		return nil, fmt.Errorf("%s: %w", op, ErrInvalidEmail)
	}

	password = normalizePassword(password)
	if !isPasswordValid(password) {
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

func (s *Service) Login(ctx context.Context, email string, password string) (*User, error) {
	const op = "users.service.Login"

	email = normalizeEmail(email)
	password = normalizePassword(password)

	if !isEmailValid(email) || !isPasswordValid(password) {
		return nil, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := s.hasher.Compare(user.PasswordHash, password); err != nil {
		return nil, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	return user, nil
}

func normalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

func normalizePassword(password string) string {
	return strings.TrimSpace(password)
}

func isEmailValid(email string) bool {
	return email != "" && strings.Contains(email, "@")
}

func isPasswordValid(password string) bool {
	return len(password) >= 8
}
