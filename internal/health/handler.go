package health

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/MSNZT/orderflow/internal/httpresponse"
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
	_ = httpresponse.JSON(w, http.StatusOK, httpresponse.StatusResponse{
		Status: httpresponse.StatusOK,
	})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()

	if err := h.dbPool.Ping(ctx); err != nil {
		h.log.Warn("postgres readiness check failed", slog.String("error", err.Error()))
		_ = httpresponse.JSON(w, http.StatusServiceUnavailable, httpresponse.StatusResponse{
			Status: httpresponse.StatusError,
		})
		return
	}

	_ = httpresponse.JSON(w, http.StatusOK, httpresponse.StatusResponse{
		Status: httpresponse.StatusOK,
	})
}
