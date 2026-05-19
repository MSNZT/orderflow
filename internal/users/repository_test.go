package users

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepository_CreateAndGetByEmail(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t)

	repo := NewRepository(pool)
	user := User{
		ID:           uuid.New(),
		Email:        "test-" + uuid.NewString() + "@mail.com",
		PasswordHash: "hash",
		Role:         RoleCustomer,
	}

	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	got, err := repo.GetByEmail(ctx, user.Email)
	if err != nil {
		t.Fatalf("get user by email: %v", err)
	}

	if got.ID != user.ID {
		t.Fatalf("expected id: %s, got: %v", user.ID, got.ID)
	}

	if got.PasswordHash != user.PasswordHash {
		t.Fatalf("expected password_hash: %s, got: %v", user.PasswordHash, got.PasswordHash)
	}

	if got.Role != user.Role {
		t.Fatalf("expected role: %s, got: %v", user.Role, got.Role)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID)
	})
}

func TestRepository_CreateAndGetByID(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t)

	repo := NewRepository(pool)
	user := User{
		ID:           uuid.New(),
		Email:        "test-" + uuid.NewString() + "@mail.com",
		PasswordHash: "hash",
		Role:         RoleCustomer,
	}

	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	got, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("get user by id: %v", err)
	}

	if got.ID != user.ID {
		t.Fatalf("expected id: %s, got: %v", user.ID, got.ID)
	}

	if got.PasswordHash != user.PasswordHash {
		t.Fatalf("expected password_hash: %s, got: %v", user.PasswordHash, got.PasswordHash)
	}

	if got.Role != user.Role {
		t.Fatalf("expected role: %s, got: %v", user.Role, got.Role)
	}

	if got.Email != user.Email {
		t.Fatalf("expected email %s, got %s", user.Email, got.Email)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID)
	})
}

func TestRepository_GetByEmail_NotFound(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t)
	repo := NewRepository(pool)

	email := "missing-" + uuid.NewString() + "@mail.com"

	_, err := repo.GetByEmail(ctx, email)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t)
	repo := NewRepository(pool)

	id := uuid.New()

	_, err := repo.GetByID(ctx, id)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestRepository_Create_DuplicateEmail(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t)
	repo := NewRepository(pool)

	email := "duplicate-" + uuid.NewString() + "@mail.com"

	first := User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: "hash",
		Role:         RoleCustomer,
	}

	second := User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: "hash",
		Role:         RoleCustomer,
	}

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("create first user: %v", err)
	}

	err := repo.Create(ctx, second)
	if err == nil {
		t.Fatalf("expected err, got nil")
	}

	if !errors.Is(err, ErrEmailAlreadyUsed) {
		t.Fatalf("expected ErrEmailAlreadyUsed, got %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", first.ID)
	})
}

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://orderflow:orderflow@localhost:5432/orderflow?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("create pg pool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping postgres: %v", err)
	}

	t.Cleanup(pool.Close)

	return pool
}
