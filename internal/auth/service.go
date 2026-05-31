package auth

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/MSNZT/orderflow/internal/sessions"
	"github.com/MSNZT/orderflow/internal/token"
	"github.com/MSNZT/orderflow/internal/users"
	"github.com/google/uuid"
)

type TokenManager interface {
	GenerateAccessToken(userID uuid.UUID, role users.Role) (string, error)
	AccessTTL() time.Duration
}

type Service struct {
	usersService       *users.Service
	tokenManager       TokenManager
	sessionsRepository *sessions.Repository
	refreshTTL         time.Duration
}

type LoginResult struct {
	User            *users.User
	AccessToken     string
	RefreshToken    string
	AccessTokenTTL  time.Duration
	RefreshTokenTLL time.Duration
}

func NewService(usersService *users.Service, tokenManager TokenManager, sessionsRepository *sessions.Repository, refreshTTL time.Duration) *Service {
	return &Service{
		usersService:       usersService,
		tokenManager:       tokenManager,
		sessionsRepository: sessionsRepository,
		refreshTTL:         refreshTTL,
	}
}

func (s *Service) Login(ctx context.Context, email string, password string, userAgent string, ipAddress *net.IP) (*LoginResult, error) {
	const op = "auth.service.Login"

	user, err := s.usersService.Login(ctx, email, password)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	accessToken, err := s.tokenManager.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	refreshToken, err := token.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	refreshTokenHash := token.HashRefreshToken(refreshToken)

	err = s.sessionsRepository.Create(ctx, sessions.Session{
		ID:               uuid.New(),
		UserID:           user.ID,
		RefreshTokenHash: refreshTokenHash,
		UserAgent:        &userAgent,
		IPAddress:        ipAddress,
		ExpiresAt:        time.Now().Add(s.refreshTTL),
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &LoginResult{
		User:            user,
		AccessToken:     accessToken,
		RefreshToken:    refreshToken,
		AccessTokenTTL:  s.tokenManager.AccessTTL(),
		RefreshTokenTLL: s.refreshTTL,
	}, nil
}
