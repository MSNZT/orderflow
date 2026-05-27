package authcontext

import (
	"context"

	"github.com/MSNZT/orderflow/internal/users"
	"github.com/google/uuid"
)

type contextKey = string

const (
	userIDKey   contextKey = "user_id"
	userRoleKey contextKey = "user_role"
)

func WithUser(ctx context.Context, userID uuid.UUID, role users.Role) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, userRoleKey, role)
	return ctx
}

func UserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	return userID, ok
}

func UserRole(ctx context.Context) (users.Role, bool) {
	role, ok := ctx.Value(userRoleKey).(users.Role)
	return role, ok
}
