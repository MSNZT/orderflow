package health

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/MSNZT/orderflow/internal/transport/http/response"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	log    *slog.Logger
	dbPool *pgxpool.Pool
}

func NewHandler(log *slog.Logger, dbPool *pgxpool.Pool) *Handler {
	return &Handler{log: log, dbPool: dbPool}
}

func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	_ = response.JSON(w, http.StatusOK, response.StatusResponse{
		Status: response.StatusOK,
	})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()

	if err := h.dbPool.Ping(ctx); err != nil {
		h.log.Warn("postgres readiness check failed", slog.String("error", err.Error()))
		_ = response.JSON(w, http.StatusServiceUnavailable, response.StatusResponse{
			Status: response.StatusError,
		})
		return
	}

	_ = response.JSON(w, http.StatusOK, response.StatusResponse{
		Status: response.StatusOK,
	})
}
