package router

import (
	"log/slog"
	"net/http"

	"github.com/MSNZT/orderflow/internal/auth"
	"github.com/MSNZT/orderflow/internal/health"
	"github.com/MSNZT/orderflow/internal/httpmiddleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(log *slog.Logger, authHandler *auth.Handler, healthHandler *health.Handler, tokenParser httpmiddleware.TokenParser) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(httpmiddleware.RequestLogger(log))
	r.Use(middleware.Recoverer)

	r.Get("/health/live", healthHandler.Live)
	r.Get("/health/ready", healthHandler.Ready)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/logout", authHandler.Logout)
		r.Post("/auth/refresh", authHandler.Refresh)

		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.Auth(tokenParser))
			r.Get("/me", authHandler.Me)
		})

	})
	return r
}
