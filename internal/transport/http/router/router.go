package router

import (
	"log/slog"
	"net/http"

	"github.com/MSNZT/orderflow/internal/app/users"
	"github.com/MSNZT/orderflow/internal/transport/http/auth"
	"github.com/MSNZT/orderflow/internal/transport/http/cart"
	"github.com/MSNZT/orderflow/internal/transport/http/health"
	"github.com/MSNZT/orderflow/internal/transport/http/httpmw"
	"github.com/MSNZT/orderflow/internal/transport/http/orders"
	"github.com/MSNZT/orderflow/internal/transport/http/payments"
	"github.com/MSNZT/orderflow/internal/transport/http/products"
	"github.com/MSNZT/orderflow/internal/transport/http/response"
	"github.com/MSNZT/orderflow/internal/transport/http/webhooks"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type RouterDependencies struct {
	AuthHandler            *auth.Handler
	ProductsHandler        *products.Handler
	CartHandler            *cart.Handler
	HealthHandler          *health.Handler
	OrderHandler           *orders.Handler
	PaymentHandler         *payments.Handler
	WebhookHandler         *webhooks.Handler
	MetricsHandler         http.Handler
	RequestMetricsRecorder httpmw.RequestMetricsRecorder
}

func NewRouter(
	log *slog.Logger, resp *response.Response, tokenParser httpmw.TokenParser, deps RouterDependencies,
) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(httpmw.RequestLogger(log))
	r.Use(
		httpmw.RequestMetrics(deps.RequestMetricsRecorder),
	)

	mw := httpmw.NewHandlerWrapper(log, resp)
	w := mw.Wrap

	r.Get("/health/live", w(deps.HealthHandler.Live))
	r.Get("/health/ready", w(deps.HealthHandler.Ready))
	r.Method(http.MethodGet, "/metrics", deps.MetricsHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", w(deps.AuthHandler.Register))
		r.Post("/auth/login", w(deps.AuthHandler.Login))
		r.Post("/auth/logout", w(deps.AuthHandler.Logout))
		r.Post("/auth/refresh", w(deps.AuthHandler.Refresh))

		r.Group(func(r chi.Router) {
			r.Use(httpmw.Auth(tokenParser, resp))
			r.Get("/me", w(deps.AuthHandler.Me))
		})

		r.Route("/products", func(r chi.Router) {
			r.Get("/", w(deps.ProductsHandler.List))
			r.Get("/{id}", w(deps.ProductsHandler.GetByID))
		})

		r.Route("/management/products", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(httpmw.Auth(tokenParser, resp))
				r.Use(httpmw.RequireRole(resp, users.RoleManager, users.RoleAdmin))

				r.Post("/", w(deps.ProductsHandler.Create))
			})
		})

		r.Route("/cart", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(httpmw.Auth(tokenParser, resp))

				r.Get("/", w(deps.CartHandler.GetItems))
				r.Post("/items", w(deps.CartHandler.AddItem))
				r.Patch("/items/{productID}", w(deps.CartHandler.UpdateItemQuantity))
				r.Delete("/items/{productID}", w(deps.CartHandler.DeleteItem))
				r.Delete("/items", w(deps.CartHandler.ClearItems))
			})
		})

		r.Route("/orders", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(httpmw.Auth(tokenParser, resp))
				r.Get("/", w(deps.OrderHandler.ListByUserID))
				r.Get("/{orderID}", w(deps.OrderHandler.GetByID))
				r.Post("/", w(deps.OrderHandler.CreateOrder))

				r.Post("/{orderID}/payments", w(deps.PaymentHandler.CreatePayment))
			})
		})

		r.Post("/webhooks/yookassa", w(deps.WebhookHandler.YooKassa))

	})
	return r
}
