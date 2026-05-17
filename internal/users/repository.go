package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrEmailAlreadyUsed = errors.New("user already used")
)

func (r *Repository) Create(ctx context.Context, user User) error {
	const op = "users.repository.Create"

	query := `INSERT INTO users(id, email, password_hash, role) 
	VALUES ($1, $2, $3, $4)`

	var pgErr *pgconn.PgError

	_, err := r.pool.Exec(ctx, query, user.ID, user.Email, user.PasswordHash, user.Role)
	if err != nil {
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%v: %w", op, ErrEmailAlreadyUsed)
		}

		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	const op = "users.repository.GetByEmail"

	query := "SELECT * FROM users WHERE email = $1"
	var user User

	err := r.pool.QueryRow(ctx, query, email).Scan(&user)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%v: %w", op, ErrUserNotFound)
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &user, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	const op = "users.repository.GetByID"

	query := "SELECT * FROM users WHERE id = $1"

	var user User

	err := r.pool.QueryRow(ctx, query, id).Scan(&user)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%v: %w", op, ErrUserNotFound)
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &user, nil
}
