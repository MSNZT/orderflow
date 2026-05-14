package router

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(dbPool *pgxpool.Pool, log *slog.Logger) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	h := NewHandler(dbPool, log)

	r.Get("/health/live", h.HealthLive)
	r.Get("/health/ready", h.HealthReady)
	return r
}
