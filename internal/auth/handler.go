package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/MSNZT/orderflow/internal/httpresponse"
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

type Handler struct {
	log          *slog.Logger
	usersService *users.Service
}

func NewHandler(log *slog.Logger, usersService *users.Service) *Handler {
	return &Handler{log: log, usersService: usersService}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	const op = "auth.handler.Register"

	var req registerRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		err := httpresponse.JSON(w, http.StatusBadRequest, httpresponse.StatusResponse{
			Status:  "error",
			Message: "invalid request body",
		})

		if err != nil {
			httpresponse.Error(w, http.StatusInternalServerError, "internal server error")
		}

		return
	}

	user, err := h.usersService.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		h.log.Error("failed to register user", slog.String("op", op), slog.String("error", err.Error()))
		mapError(err, w)
		return
	}

	res := registerResponse{
		ID:    user.ID.String(),
		Email: user.Email,
		Role:  user.Role,
	}

	if err := httpresponse.JSON(w, http.StatusOK, res); err != nil {
		h.log.Error("failed to send register response", slog.String("op", op), slog.String("err", err.Error()))
		httpresponse.Error(w, http.StatusInternalServerError, "internal server error")
	}
}

func mapError(err error, w http.ResponseWriter) {
	switch {
	case errors.Is(err, users.ErrInvalidEmail):
		_ = httpresponse.Error(w, http.StatusBadRequest, "invalid email")
		return
	case errors.Is(err, users.ErrInvalidPassword):
		_ = httpresponse.Error(w, http.StatusBadRequest, "invalid password")
		return
	case errors.Is(err, users.ErrEmailAlreadyUsed):
		_ = httpresponse.Error(w, http.StatusConflict, "email already used")
	default:
		_ = httpresponse.Error(w, http.StatusInternalServerError, "internal server error")
	}
}
