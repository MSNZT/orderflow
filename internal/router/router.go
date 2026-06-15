package router

import (
	"log/slog"
	"net/http"

	"github.com/MSNZT/orderflow/internal/auth"
	"github.com/MSNZT/orderflow/internal/cart"
	"github.com/MSNZT/orderflow/internal/health"
	"github.com/MSNZT/orderflow/internal/httpmiddleware"
	"github.com/MSNZT/orderflow/internal/products"
	"github.com/MSNZT/orderflow/internal/users"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type RouterDependencies struct {
	AuthHandler     *auth.Handler
	ProductsHandler *products.Handler
	CartHandler     *cart.Handler
	HealthHandler   *health.Handler
}

func NewRouter(log *slog.Logger, tokenParser httpmiddleware.TokenParser, deps RouterDependencies) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(httpmiddleware.RequestLogger(log))
	r.Use(middleware.Recoverer)

	r.Get("/health/live", deps.HealthHandler.Live)
	r.Get("/health/ready", deps.HealthHandler.Ready)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", deps.AuthHandler.Register)
		r.Post("/auth/login", deps.AuthHandler.Login)
		r.Post("/auth/logout", deps.AuthHandler.Logout)
		r.Post("/auth/refresh", deps.AuthHandler.Refresh)

		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.Auth(tokenParser))
			r.Get("/me", deps.AuthHandler.Me)
		})

		r.Route("/products", func(r chi.Router) {
			r.Get("/", deps.ProductsHandler.List)
			r.Get("/{id}", deps.ProductsHandler.GetByID)
		})

		r.Route("/management/products", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(httpmiddleware.Auth(tokenParser))
				r.Use(httpmiddleware.RequireRole(users.RoleManager, users.RoleAdmin))

				r.Post("/", deps.ProductsHandler.Create)
			})
		})

		r.Route("/cart", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(httpmiddleware.Auth(tokenParser))

				r.Get("/", deps.CartHandler.GetItems)
				r.Post("/items", deps.CartHandler.AddItem)
				r.Patch("/items/{productID}", deps.CartHandler.UpdateItemQuantity)
				r.Delete("/items/{productID}", deps.CartHandler.DeleteItem)
				r.Delete("/items", deps.CartHandler.ClearItems)
			})
		})
	})
	return r
}
