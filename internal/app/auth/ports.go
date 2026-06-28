package auth

import (
	"context"
	"time"

	"github.com/MSNZT/orderflow/internal/app/users"
	"github.com/google/uuid"
)

type UsersService interface {
	Login(
		ctx context.Context,
		email string,
		password string,
	) (*users.User, error)

	GetByID(
		ctx context.Context,
		userID uuid.UUID,
	) (*users.User, error)
}

type TokenManager interface {
	GenerateAccessToken(
		userID uuid.UUID,
		role users.Role,
	) (string, error)

	AccessTTL() time.Duration

	GenerateRefreshToken() (string, error)
	HashRefreshToken(token string) string
}
