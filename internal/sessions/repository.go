package sessions

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, s Session) error {
	const op = "sessions.repository.Create"

	query := `INSERT INTO user_sessions (
		id,
		user_id,
		refresh_token_hash,
		user_agent,
		ip_address,
		expires_at
	) VALUES ($1, $2, $3, $4, $5, $6);`

	_, err := r.pool.Exec(ctx, query,
		s.ID, s.UserID, s.RefreshTokenHash, s.UserAgent, s.IPAddress, s.ExpiresAt)

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Repository) FindByRefreshTokenHash(ctx context.Context, tokenHash string) (*Session, error) {
	const op = "sessions.repository.FindByRefreshTokenHash"

	query := `SELECT
		id,
		user_id,
		refresh_token_hash,
		user_agent,
		ip_address,
		created_at,
		updated_at,
		last_used_at,
		expires_at,
		revoked_at,
	FROM user_sessions 
	WHERE refresh_token_hash = $1
	);`

	var session *Session

	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(&session)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, ErrSessionNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return session, nil
}

func (r *Repository) UpdateHash(ctx context.Context, id uuid.UUID, tokenHash string) error {
	const op = "sessions.repository.UpdateSession"

	query := `UPDATE user_sessions SET refresh_token_hash = $2 WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query, id, tokenHash)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, ErrSessionNotFound)
	}

	return nil
}

func (r *Repository) Revoke(ctx context.Context, tokenHash string) error {
	const op = "sessions.repository.Revoke"

	query := `UPDATE user_sessions 
			  SET revoked_at = NOW() 
			  WHERE refresh_token_hash = $1 AND revoked_at IS NULL;`

	tag, err := r.pool.Exec(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, ErrSessionNotFound)
	}

	return nil
}
