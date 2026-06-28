package sessions

import (
	"context"
	"errors"
	"fmt"
	"time"

	sessionsapp "github.com/MSNZT/orderflow/internal/app/sessions"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

var _ sessionsapp.Repository = (*Repository)(nil)

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool: pool,
	}
}

func (r *Repository) Create(
	ctx context.Context,
	session sessionsapp.Session,
) error {
	const op = "postgres.sessions.Repository.Create"

	query := `
		INSERT INTO user_sessions (
			id,
			user_id,
			refresh_token_hash,
			user_agent,
			ip_address,
			expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6);
	`

	_, err := r.pool.Exec(
		ctx,
		query,
		session.ID,
		session.UserID,
		session.RefreshTokenHash,
		session.UserAgent,
		session.IPAddress,
		session.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Repository) FindByRefreshTokenHash(
	ctx context.Context,
	tokenHash string,
) (*sessionsapp.Session, error) {
	const op = "postgres.sessions.Repository.FindByRefreshTokenHash"

	query := `
		SELECT
			id,
			user_id,
			refresh_token_hash,
			user_agent,
			ip_address,
			created_at,
			updated_at,
			last_used_at,
			expires_at,
			revoked_at
		FROM user_sessions
		WHERE refresh_token_hash = $1;
	`

	var session sessionsapp.Session

	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshTokenHash,
		&session.UserAgent,
		&session.IPAddress,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.LastUsedAt,
		&session.ExpiresAt,
		&session.RevokedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf(
				"%s: %w",
				op,
				sessionsapp.ErrSessionNotFound,
			)
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &session, nil
}

func (r *Repository) RotateRefreshToken(
	ctx context.Context,
	id uuid.UUID,
	tokenHash string,
	expiresAt time.Time,
) error {
	const op = "postgres.sessions.Repository.RotateRefreshToken"

	query := `
		UPDATE user_sessions
		SET
			refresh_token_hash = $2,
			expires_at = $3,
			last_used_at = NOW(),
			updated_at = NOW()
		WHERE id = $1;
	`

	tag, err := r.pool.Exec(
		ctx,
		query,
		id,
		tokenHash,
		expiresAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf(
			"%s: %w",
			op,
			sessionsapp.ErrSessionNotFound,
		)
	}

	return nil
}

func (r *Repository) Revoke(
	ctx context.Context,
	tokenHash string,
) error {
	const op = "postgres.sessions.Repository.Revoke"

	query := `
		UPDATE user_sessions
		SET
			revoked_at = COALESCE(revoked_at, NOW()),
			updated_at = CASE
				WHEN revoked_at IS NULL THEN NOW()
				ELSE updated_at
			END
		WHERE refresh_token_hash = $1;
	`

	tag, err := r.pool.Exec(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf(
			"%s: %w",
			op,
			sessionsapp.ErrSessionNotFound,
		)
	}

	return nil
}
