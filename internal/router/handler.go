package router

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	dbPool *pgxpool.Pool
	log    *slog.Logger
}

type HealthResponse struct {
	Status string `json:"status"`
}

func NewHandler(dbPool *pgxpool.Pool, log *slog.Logger) *Handler {
	return &Handler{dbPool: dbPool, log: log}
}

const (
	StatusOK = "ok"
)

func (h *Handler) HealthLive(w http.ResponseWriter, r *http.Request) {
	_ = WriteJSON(w, http.StatusOK, HealthResponse{
		Status: StatusOK,
	})
}

func (h *Handler) HealthReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()

	if err := h.dbPool.Ping(ctx); err != nil {
		h.log.Warn("postgres readiness check failed", slog.String("error", err.Error()))
		_ = WriteJSON(w, http.StatusServiceUnavailable, HealthResponse{
			Status: "error",
		})
		return
	}

	_ = WriteJSON(w, http.StatusOK, HealthResponse{
		Status: StatusOK,
	})
}

func WriteJSON(w http.ResponseWriter, statusCode int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(v)
}
