package users

import (
	"context"
	"errors"
	"fmt"

	usersapp "github.com/MSNZT/orderflow/internal/app/users"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

var _ usersapp.Repository = (*Repository)(nil)

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, user usersapp.User) error {
	const op = "users.repository.Create"

	query := `INSERT INTO users(id, email, password_hash, role) 
	VALUES ($1, $2, $3, $4)`

	var pgErr *pgconn.PgError

	_, err := r.pool.Exec(ctx, query, user.ID, user.Email, user.PasswordHash, user.Role)
	if err != nil {
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "users_email_key" {
				return fmt.Errorf("%s: %w", op, usersapp.ErrEmailAlreadyUsed)
			}
		}

		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*usersapp.User, error) {
	const op = "users.repository.GetByEmail"

	query := `
		SELECT 
			id, 
			email, 
			password_hash, 
			role, 
			created_at, 
			updated_at 
		FROM users
		WHERE email = $1`
	var u usersapp.User

	err := r.pool.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, usersapp.ErrUserNotFound)
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &u, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*usersapp.User, error) {
	const op = "users.repository.GetByID"

	query := `
		SELECT 
			id, 
			email, 
			password_hash, 
			role, 
			created_at, 
			updated_at 
		FROM users 
		WHERE id = $1`

	var u usersapp.User

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, usersapp.ErrUserNotFound)
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &u, nil
}
