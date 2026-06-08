package sessions

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/MSNZT/orderflow/internal/users"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepository_Create(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t)
	sessionRepository := NewRepository(pool)
	user := createTestUser(t, ctx, pool)

	t.Cleanup(func() {
		deleteTestUser(t, pool, ctx, user.ID)
	})
	session := createTestSession(t, user.ID, ctx, sessionRepository)

	s, err := sessionRepository.FindByRefreshTokenHash(ctx, session.RefreshTokenHash)
	if err != nil {
		t.Fatalf("failed to find session by refresh token hash: %v", err)
	}

	if s.ID != session.ID {
		t.Fatalf("expected session id: %s, got: %s", session.ID, s.ID)
	}

	if s.UserID != user.ID {
		t.Fatalf("expected user id: %s, got: %s", user.ID, s.UserID)
	}

	if s.RefreshTokenHash != session.RefreshTokenHash {
		t.Fatalf("expected refresh token hash: %s, got: %s", session.RefreshTokenHash, s.RefreshTokenHash)
	}

	if !s.ExpiresAt.Equal(session.ExpiresAt) {
		t.Fatalf("expected expires at: %s, got: %s", session.ExpiresAt, s.ExpiresAt)
	}

	if s.RevokedAt != nil {
		t.Fatalf("expected revoked at nil, got: %s", s.RevokedAt)
	}
}

func TestRepository_FindByRefreshTokenHash_NotFound(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t)
	sessionRepository := NewRepository(pool)

	refreshTokenHash := "not_found-token-hash" + newTestUUID(t).String()

	_, err := sessionRepository.FindByRefreshTokenHash(ctx, refreshTokenHash)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got: %v", err)
	}
}

func TestRepository_RotateRefreshToken(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t)
	sessionRepository := NewRepository(pool)
	user := createTestUser(t, ctx, pool)

	t.Cleanup(func() {
		deleteTestUser(t, pool, ctx, user.ID)
	})

	oldSession := createTestSession(t, user.ID, ctx, sessionRepository)
	time.Sleep(10 * time.Millisecond)

	newRefreshTokenHash := "new_refresh_token-hash" + newTestUUID(t).String()
	expiresAt := time.Now().UTC().Truncate(time.Microsecond).Add(168 * time.Hour)

	err := sessionRepository.RotateRefreshToken(ctx, oldSession.ID, newRefreshTokenHash, expiresAt)
	if err != nil {
		t.Fatalf("failed to rotate refresh token: %v", err)
	}

	s, err := sessionRepository.FindByRefreshTokenHash(ctx, newRefreshTokenHash)
	if err != nil {
		t.Fatalf("failed to find rotated session: %v", err)
	}

	if s.ID != oldSession.ID {
		t.Fatalf("expected session id: %s, got: %s", oldSession.ID, s.ID)
	}

	if s.RefreshTokenHash != newRefreshTokenHash {
		t.Fatalf("expected refresh token hash: %s, got: %s", newRefreshTokenHash, s.RefreshTokenHash)
	}

	if !s.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("expected expires at: %v, got: %v", expiresAt, s.ExpiresAt)
	}

	if !s.UpdatedAt.After(oldSession.UpdatedAt) {
		t.Fatalf("expected updated_at to increase, old: %v, new: %v", oldSession.UpdatedAt, s.UpdatedAt)
	}

	if !s.LastUsedAt.After(oldSession.LastUsedAt) {
		t.Fatalf("expected last_used_at to increase, old: %v, new: %v", oldSession.LastUsedAt, s.LastUsedAt)
	}

	_, err = sessionRepository.FindByRefreshTokenHash(ctx, oldSession.RefreshTokenHash)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got: %v", err)
	}
}

func TestRepository_RotateRefreshToken_NotFound(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t)
	sessionRepository := NewRepository(pool)

	id := newTestUUID(t)

	newRefreshTokenHash := "new_refresh_token-hash" + id.String()
	expiresAt := time.Now().UTC().Truncate(time.Microsecond).Add(168 * time.Hour)

	err := sessionRepository.RotateRefreshToken(ctx, id, newRefreshTokenHash, expiresAt)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got: %v", err)
	}
}

func TestRepository_RevokeRefreshToken(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t)
	sessionRepository := NewRepository(pool)
	user := createTestUser(t, ctx, pool)

	t.Cleanup(func() {
		deleteTestUser(t, pool, ctx, user.ID)
	})

	oldSession := createTestSession(t, user.ID, ctx, sessionRepository)
	time.Sleep(10 * time.Millisecond)

	err := sessionRepository.Revoke(ctx, oldSession.RefreshTokenHash)
	if err != nil {
		t.Fatalf("failed to revoke refresh token: %v", err)
	}

	s, err := sessionRepository.FindByRefreshTokenHash(ctx, oldSession.RefreshTokenHash)
	if err != nil {
		t.Fatalf("failed to find session: %v", err)
	}

	if s.ID != oldSession.ID {
		t.Fatalf("expected session id: %s, got: %s", oldSession.ID, s.ID)
	}

	if s.RevokedAt == nil {
		t.Fatalf("expected revoked_at to be set, got nil")
	}

	if !s.UpdatedAt.After(oldSession.UpdatedAt) {
		t.Fatalf("expected updated_at to increase, old: %v, new: %v", oldSession.UpdatedAt, s.UpdatedAt)
	}
}

func TestRepository_RevokeRefreshToken_Idempotent(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t)
	sessionRepository := NewRepository(pool)
	user := createTestUser(t, ctx, pool)

	t.Cleanup(func() {
		deleteTestUser(t, pool, ctx, user.ID)
	})

	session := createTestSession(t, user.ID, ctx, sessionRepository)
	time.Sleep(10 * time.Millisecond)

	err := sessionRepository.Revoke(ctx, session.RefreshTokenHash)
	if err != nil {
		t.Fatalf("failed to revoke refresh token: %v", err)
	}

	s, err := sessionRepository.FindByRefreshTokenHash(ctx, session.RefreshTokenHash)
	if err != nil {
		t.Fatalf("failed to find session: %v", err)
	}

	if s.RevokedAt == nil {
		t.Fatalf("expected revoked_at must be set, got nil")
	}

	if s.ID != session.ID {
		t.Fatalf("expected session id: %s, got: %s", session.ID, s.ID)
	}

	time.Sleep(10 * time.Millisecond)
	err = sessionRepository.Revoke(ctx, session.RefreshTokenHash)
	if err != nil {
		t.Fatalf("failed to revoke refresh token: %v", err)
	}

	s2, err := sessionRepository.FindByRefreshTokenHash(ctx, session.RefreshTokenHash)
	if err != nil {
		t.Fatalf("failed to find session: %v", err)
	}

	if s2.RevokedAt == nil {
		t.Fatalf("expected revoked_at must be set, got nil")
	}

	if !s2.RevokedAt.Equal(*s.RevokedAt) {
		t.Fatalf("revoked at must be equal, old: %v, new: %v", s.RevokedAt, s2.RevokedAt)
	}

	if !s2.UpdatedAt.Equal(s.UpdatedAt) {
		t.Fatalf("expected updated_at to be equal, old: %v, new: %v", s.UpdatedAt, s2.UpdatedAt)
	}
}

func TestRepository_RevokeRefreshToken_NotFound(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t)
	sessionRepository := NewRepository(pool)

	missingHash := "missing-hash" + newTestUUID(t).String()

	err := sessionRepository.Revoke(ctx, missingHash)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got: %v", err)
	}
}

func createTestUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool) *users.User {
	t.Helper()

	repo := users.NewRepository(pool)

	id := newTestUUID(t)

	user := users.User{
		ID:           id,
		Email:        "email" + id.String(),
		PasswordHash: "hash-password" + id.String(),
		Role:         users.RoleCustomer,
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	return &user
}

func deleteTestUser(t *testing.T, pool *pgxpool.Pool, ctx context.Context, userId uuid.UUID) {
	t.Helper()

	_, err := pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userId)
	if err != nil {
		t.Fatalf("failed to delete test user: %v", err)
	}
}

func createTestSession(t *testing.T, userId uuid.UUID, ctx context.Context, sessionRepository *Repository) *Session {
	t.Helper()

	id := newTestUUID(t)
	refreshTokenHash := "refresh-token-hash-" + id.String()
	expiresAt := time.Now().UTC().Truncate(time.Microsecond).Add(168 * time.Hour)

	session := Session{
		ID:               id,
		UserID:           userId,
		RefreshTokenHash: refreshTokenHash,
		UserAgent:        nil,
		IPAddress:        nil,
		ExpiresAt:        expiresAt,
	}

	err := sessionRepository.Create(ctx, session)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	s, err := sessionRepository.FindByRefreshTokenHash(ctx, session.RefreshTokenHash)
	if err != nil {
		t.Fatalf("failed to find by created session: %v", err)
	}

	return s
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

func newTestUUID(t *testing.T) uuid.UUID {
	t.Helper()

	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("failed to generate uuid: %v", err)
	}

	return id
}
