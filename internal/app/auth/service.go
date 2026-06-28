package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/MSNZT/orderflow/internal/app/sessions"
	"github.com/MSNZT/orderflow/internal/app/users"
	"github.com/google/uuid"
)

type TokenManager interface {
	GenerateAccessToken(userID uuid.UUID, role users.Role) (string, error)
	AccessTTL() time.Duration

	GenerateRefreshToken() (string, error)
	HashRefreshToken(token string) string
}

type Service struct {
	usersService       *users.Service
	tokenManager       TokenManager
	sessionsRepository sessions.Repository
	refreshTTL         time.Duration
}

type LoginResult struct {
	User            *users.User
	AccessToken     string
	RefreshToken    string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type RefreshResult struct {
	AccessToken     string
	RefreshToken    string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

func NewService(usersService *users.Service, tokenManager TokenManager, sessionsRepository sessions.Repository, refreshTTL time.Duration) *Service {
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

	refreshToken, err := s.tokenManager.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	refreshTokenHash := s.tokenManager.HashRefreshToken(refreshToken)

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	err = s.sessionsRepository.Create(ctx, sessions.Session{
		ID:               id,
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
		RefreshTokenTTL: s.refreshTTL,
	}, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*RefreshResult, error) {
	const op = "auth.service.Refresh"
	hashRefreshToken := s.tokenManager.HashRefreshToken(refreshToken)
	session, err := s.sessionsRepository.FindByRefreshTokenHash(ctx, hashRefreshToken)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if session.RevokedAt != nil {
		return nil, fmt.Errorf("%s: %w", op, sessions.ErrSessionRevoked)
	}

	now := time.Now()

	if !session.ExpiresAt.After(now) {
		return nil, fmt.Errorf("%s: %w", op, sessions.ErrSessionExpired)
	}

	user, err := s.usersService.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	accessToken, err := s.tokenManager.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	newRefreshToken, err := s.tokenManager.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	hashRefreshToken = s.tokenManager.HashRefreshToken(newRefreshToken)

	err = s.sessionsRepository.RotateRefreshToken(ctx, session.ID, hashRefreshToken, now.Add(s.refreshTTL))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &RefreshResult{
		AccessToken:     accessToken,
		RefreshToken:    newRefreshToken,
		AccessTokenTTL:  s.tokenManager.AccessTTL(),
		RefreshTokenTTL: s.refreshTTL,
	}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	const op = "auth.service.Logout"
	tokenHash := s.tokenManager.HashRefreshToken(refreshToken)

	if err := s.sessionsRepository.Revoke(ctx, tokenHash); err != nil {
		if errors.Is(err, sessions.ErrSessionNotFound) {
			return nil
		}

		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
