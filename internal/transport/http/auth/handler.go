package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"

	authapp "github.com/MSNZT/orderflow/internal/app/auth"
	"github.com/MSNZT/orderflow/internal/app/sessions"
	"github.com/MSNZT/orderflow/internal/app/users"
	"github.com/MSNZT/orderflow/internal/transport/http/authcontext"
	"github.com/MSNZT/orderflow/internal/transport/http/httpmw"
	"github.com/MSNZT/orderflow/internal/transport/http/response"
)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerResponse struct {
	ID    string     `json:"id"`
	Email string     `json:"email"`
	Role  users.Role `json:"role"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string       `json:"access_token"`
	ExpiresIn   int          `json:"expires_in"`
	User        userResponse `json:"user"`
}

type refreshResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type userResponse struct {
	ID    string     `json:"id"`
	Email string     `json:"email"`
	Role  users.Role `json:"role"`
}

type Service interface {
	Login(ctx context.Context, email string, password string, userAgent string, ipAddress *net.IP) (*authapp.LoginResult, error)
	Refresh(ctx context.Context, refreshToken string) (*authapp.RefreshResult, error)
	Logout(ctx context.Context, refreshToken string) error
}

type Handler struct {
	resp         *response.Response
	usersService *users.Service
	authService  Service
	authManager  *CookieManager
}

func NewHandler(
	resp *response.Response, usersService *users.Service, authService Service, authManager *CookieManager,
) *Handler {
	return &Handler{resp: resp, usersService: usersService, authService: authService, authManager: authManager}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) error {
	const op = "auth.handler.Register"

	var req registerRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid request body")
	}

	user, err := h.usersService.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, users.ErrInvalidEmail):
			return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid email")
		case errors.Is(err, users.ErrInvalidPassword):
			return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid password")
		case errors.Is(err, users.ErrEmailAlreadyUsed):
			return httpmw.NewHTTPError(http.StatusConflict, op, "email already used")
		default:
			return fmt.Errorf("%s: failed to register user: %w", op, err)
		}
	}

	res := registerResponse{
		ID:    user.ID.String(),
		Email: user.Email,
		Role:  user.Role,
	}

	h.resp.JSON(w, http.StatusCreated, res)
	return nil
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) error {
	const op = "auth.handler.Login"

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid request body")
	}

	userAgent := r.UserAgent()

	loginResult, err := h.authService.Login(r.Context(), req.Email, req.Password, userAgent, nil)
	if err != nil {
		switch {
		case errors.Is(err, users.ErrInvalidCredentials):
			return httpmw.NewHTTPError(http.StatusUnauthorized, op, "invalid credentials")
		default:
			return fmt.Errorf("%s: failed to login user: %w", op, err)
		}
	}

	res := loginResponse{
		AccessToken: loginResult.AccessToken,
		ExpiresIn:   int(loginResult.AccessTokenTTL.Seconds()),
		User: userResponse{
			ID:    loginResult.User.ID.String(),
			Email: loginResult.User.Email,
			Role:  loginResult.User.Role,
		},
	}

	h.authManager.setRefreshToken(w, loginResult.RefreshToken, loginResult.RefreshTokenTTL)

	h.resp.JSON(w, http.StatusOK, res)
	return nil
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) error {
	const op = "auth.handler.Me"

	userID, ok := authcontext.UserID(r.Context())

	if !ok {
		return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")
	}

	user, err := h.usersService.Me(r.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, users.ErrUnauthorized):
			return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")
		default:
			return fmt.Errorf("%s: failed to get user: %w", op, err)
		}
	}

	res := userResponse{
		ID:    user.ID.String(),
		Email: user.Email,
		Role:  user.Role,
	}

	h.resp.JSON(w, http.StatusOK, res)
	return nil
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) error {
	const op = "auth.handler.Refresh"

	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")
		}

		return fmt.Errorf("%s: failed to get refresh token from cookie: %w", op, err)
	}

	if cookie.Value == "" {
		return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")
	}

	refreshResult, err := h.authService.Refresh(r.Context(), cookie.Value)
	if err != nil {
		switch {
		case errors.Is(err, sessions.ErrSessionExpired),
			errors.Is(err, sessions.ErrSessionNotFound),
			errors.Is(err, sessions.ErrSessionRevoked),
			errors.Is(err, users.ErrUserNotFound):
			h.authManager.clearRefreshCookie(w)
			return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")
		default:
			return fmt.Errorf("%s: failed to update refresh session: %w", op, err)
		}
	}

	h.authManager.setRefreshToken(w, refreshResult.RefreshToken, refreshResult.RefreshTokenTTL)
	res := refreshResponse{
		AccessToken: refreshResult.AccessToken,
		ExpiresIn:   int(refreshResult.AccessTokenTTL.Seconds()),
	}

	h.resp.JSON(w, http.StatusOK, res)
	return nil
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) error {
	const op = "auth.handler.Logout"
	cookie, err := r.Cookie(refreshCookieName)

	if err != nil || cookie.Value == "" {
		h.authManager.clearRefreshCookie(w)
		h.resp.NoContent(w)
		return nil
	}

	if err := h.authService.Logout(r.Context(), cookie.Value); err != nil {
		h.authManager.clearRefreshCookie(w)
		return fmt.Errorf("%s: failed to logout: %w", op, err)
	}

	h.authManager.clearRefreshCookie(w)
	h.resp.NoContent(w)
	return nil
}
