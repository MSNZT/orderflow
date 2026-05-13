package router

import (
	"github.com/MSNZT/orderflow/internal/router/handler"
	"github.com/go-chi/chi/v5"
)

func NewRouter() *chi.Mux {
	r := chi.NewRouter()

	h := handler.NewHandler()

	r.Get("/health/live", h.HealthLive())

	return r
}
