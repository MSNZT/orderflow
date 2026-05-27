package token

import (
	"time"

	"github.com/MSNZT/orderflow/internal/users"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Manager struct {
	secret    []byte
	accessTTL time.Duration
}

type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func NewManager(secret string, accessTTL time.Duration) *Manager {
	return &Manager{secret: []byte(secret), accessTTL: accessTTL}
}

func (m *Manager) GenerateAccessToken(userID uuid.UUID, role users.Role) (string, error) {
	now := time.Now()

	claims := Claims{
		Role: string(role),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *Manager) AccessTTL() time.Duration {
	return m.accessTTL
}
