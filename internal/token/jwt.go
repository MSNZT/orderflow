package token

import (
	"errors"
	"fmt"
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

var (
	ErrTokenExpired   = errors.New("token has expired")
	ErrTokenSignature = errors.New("invalid token signature")
	ErrTokenInvalid   = errors.New("invalid token")
)

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

func (m *Manager) ParseAccessToken(accessToken string) (*Claims, error) {
	const op = "token.jwt.ParseAccessToken"

	claims := &Claims{}
	parsedToken, err := jwt.ParseWithClaims(accessToken, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, fmt.Errorf("%s: %w", op, ErrTokenSignature)
		}
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("%s: %w", op, ErrTokenExpired)
		}

		return nil, fmt.Errorf("%s: %w", op, ErrTokenInvalid)
	}

	if !parsedToken.Valid {
		return nil, fmt.Errorf("%s: %w", op, ErrTokenInvalid)
	}

	return claims, nil
}

func (m *Manager) AccessTTL() time.Duration {
	return m.accessTTL
}
