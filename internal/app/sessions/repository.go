package sessions

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, session Session) error

	FindByRefreshTokenHash(
		ctx context.Context,
		tokenHash string,
	) (*Session, error)

	RotateRefreshToken(
		ctx context.Context,
		id uuid.UUID,
		tokenHash string,
		expiresAt time.Time,
	) error

	Revoke(ctx context.Context, tokenHash string) error
}
