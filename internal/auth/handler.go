package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/MSNZT/orderflow/internal/sessions"
	"github.com/MSNZT/orderflow/internal/transport/http/authcontext"
	"github.com/MSNZT/orderflow/internal/transport/http/response"
	"github.com/MSNZT/orderflow/internal/users"
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

type Handler struct {
	log          *slog.Logger
	usersService *users.Service
	authService  *Service
}

func NewHandler(log *slog.Logger, usersService *users.Service, authService *Service) *Handler {
	return &Handler{log: log, usersService: usersService, authService: authService}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	const op = "auth.handler.Register"

	var req registerRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.usersService.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, users.ErrInvalidEmail):
			_ = response.Error(w, http.StatusBadRequest, "invalid email")
			return
		case errors.Is(err, users.ErrInvalidPassword):
			_ = response.Error(w, http.StatusBadRequest, "invalid password")
			return
		case errors.Is(err, users.ErrEmailAlreadyUsed):
			_ = response.Error(w, http.StatusConflict, "email already used")
			return
		default:
			h.log.Error("failed to register user", slog.String("op", op), slog.String("error", err.Error()))
			_ = response.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	res := registerResponse{
		ID:    user.ID.String(),
		Email: user.Email,
		Role:  user.Role,
	}

	if err := response.JSON(w, http.StatusCreated, res); err != nil {
		h.log.Error("failed to send register response", slog.String("op", op), slog.String("error", err.Error()))
		_ = response.Error(w, http.StatusInternalServerError, "internal server error")
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	const op = "auth.handler.Login"

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userAgent := r.UserAgent()

	loginResult, err := h.authService.Login(r.Context(), req.Email, req.Password, userAgent, nil)
	if err != nil {
		switch {
		case errors.Is(err, users.ErrInvalidCredentials):
			_ = response.Error(w, http.StatusUnauthorized, "invalid credentials")
			return
		default:
			h.log.Error("failed to login user", slog.String("op", op), slog.String("error", err.Error()))
			_ = response.Error(w, http.StatusInternalServerError, "internal server error")
			return
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

	setRefreshToken(w, loginResult.RefreshToken, loginResult.RefreshTokenTTL)

	if err := response.JSON(w, http.StatusOK, res); err != nil {
		h.log.Error("failed to send login response", slog.String("op", op), slog.String("error", err.Error()))
		_ = response.Error(w, http.StatusInternalServerError, "internal server error")
	}
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	const op = "auth.handler.Me"

	userID, ok := authcontext.UserID(r.Context())

	if !ok {
		_ = response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.usersService.Me(r.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, users.ErrUnauthorized):
			_ = response.Error(w, http.StatusUnauthorized, "unauthorized")
			return
		default:
			h.log.Error("failed to get user", slog.String("op", op), slog.String("error", err.Error()))
			_ = response.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	res := userResponse{
		ID:    user.ID.String(),
		Email: user.Email,
		Role:  user.Role,
	}

	if err := response.JSON(w, http.StatusOK, res); err != nil {
		h.log.Error("failed to send me response", slog.String("op", op), slog.String("error", err.Error()))
		_ = response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	const op = "auth.handler.Refresh"

	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			_ = response.Error(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		h.log.Error("failed to get refresh token from cookie", slog.String("op", op), slog.String("error", err.Error()))
		_ = response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if cookie.Value == "" {
		_ = response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	refreshResult, err := h.authService.Refresh(r.Context(), cookie.Value)
	if err != nil {
		switch {
		case errors.Is(err, sessions.ErrSessionExpired),
			errors.Is(err, sessions.ErrSessionNotFound),
			errors.Is(err, sessions.ErrSessionRevoked),
			errors.Is(err, users.ErrUserNotFound):
			clearRefreshCookie(w)
			_ = response.Error(w, http.StatusUnauthorized, "unauthorized")
			return
		default:
			h.log.Error("failed to update refresh session", slog.String("op", op), slog.String("error", err.Error()))
			_ = response.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	setRefreshToken(w, refreshResult.RefreshToken, refreshResult.RefreshTokenTTL)
	res := refreshResponse{
		AccessToken: refreshResult.AccessToken,
		ExpiresIn:   int(refreshResult.AccessTokenTTL.Seconds()),
	}

	if err := response.JSON(w, http.StatusOK, res); err != nil {
		h.log.Error("failed to send refresh response", slog.String("op", op), slog.String("error", err.Error()))
		_ = response.Error(w, http.StatusInternalServerError, "internal server error")
	}
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	const op = "auth.handler.Logout"
	cookie, err := r.Cookie(refreshCookieName)

	if err != nil || cookie.Value == "" {
		clearRefreshCookie(w)
		response.NoContent(w)
		return
	}

	if err := h.authService.Logout(r.Context(), cookie.Value); err != nil {
		clearRefreshCookie(w)

		h.log.Error("failed to logout", slog.String("op", op), slog.String("error", err.Error()))
		_ = response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	clearRefreshCookie(w)
	response.NoContent(w)
}
