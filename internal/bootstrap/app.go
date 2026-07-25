package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	authapp "github.com/MSNZT/orderflow/internal/app/auth"
	cartapp "github.com/MSNZT/orderflow/internal/app/cart"
	"github.com/MSNZT/orderflow/internal/app/jobs"
	ordersapp "github.com/MSNZT/orderflow/internal/app/orders"
	paymentsapp "github.com/MSNZT/orderflow/internal/app/payments"
	productsapp "github.com/MSNZT/orderflow/internal/app/products"
	usersapp "github.com/MSNZT/orderflow/internal/app/users"
	"github.com/MSNZT/orderflow/internal/config"
	metricsinfra "github.com/MSNZT/orderflow/internal/infrastructure/metrics"
	"github.com/MSNZT/orderflow/internal/infrastructure/password"
	"github.com/MSNZT/orderflow/internal/infrastructure/postgres"
	cartrepo "github.com/MSNZT/orderflow/internal/infrastructure/postgres/cart"
	"github.com/MSNZT/orderflow/internal/infrastructure/postgres/inventory"
	ordersrepo "github.com/MSNZT/orderflow/internal/infrastructure/postgres/orders"
	paymentsrepo "github.com/MSNZT/orderflow/internal/infrastructure/postgres/payments"
	productsrepo "github.com/MSNZT/orderflow/internal/infrastructure/postgres/products"
	"github.com/MSNZT/orderflow/internal/infrastructure/postgres/sessions"
	usersrepo "github.com/MSNZT/orderflow/internal/infrastructure/postgres/users"
	"github.com/MSNZT/orderflow/internal/infrastructure/token"
	"github.com/MSNZT/orderflow/internal/platform/worker"
	"github.com/MSNZT/orderflow/internal/platform/yookassa"
	authhttp "github.com/MSNZT/orderflow/internal/transport/http/auth"
	carthttp "github.com/MSNZT/orderflow/internal/transport/http/cart"
	"github.com/MSNZT/orderflow/internal/transport/http/health"
	ordershttp "github.com/MSNZT/orderflow/internal/transport/http/orders"
	paymentshttp "github.com/MSNZT/orderflow/internal/transport/http/payments"
	productshttp "github.com/MSNZT/orderflow/internal/transport/http/products"
	"github.com/MSNZT/orderflow/internal/transport/http/response"
	"github.com/MSNZT/orderflow/internal/transport/http/router"
	"github.com/MSNZT/orderflow/internal/transport/http/server"
	"github.com/MSNZT/orderflow/internal/transport/http/webhooks"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	server  *server.Server
	workers *worker.Manager
	dbPool  *pgxpool.Pool
}

func New(ctx context.Context, cfg *config.Config, log *slog.Logger) (*App, error) {
	dbPool, err := postgres.NewPool(ctx, &cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	txManager := postgres.NewTxManager(dbPool)

	resp := response.New(log)
	cookieManager := authhttp.NewCookieManager(cfg.Env.IsProduction())

	healthHandler := health.NewHandler(log, resp, dbPool)

	usersRepository := usersrepo.NewRepository(dbPool)
	hasher := password.NewBcryptHasher()
	usersService := usersapp.NewService(usersRepository, hasher)
	tokenManager := token.NewManager(cfg.JWT.Secret, cfg.JWT.AccessTTL)

	yookassaClient, err := yookassa.NewClient(yookassa.YookassaClientConfig{
		APIURL:    cfg.Yookassa.APIURL,
		ShopID:    cfg.Yookassa.ShopID,
		SecretKey: cfg.Yookassa.SecretKey,
		ReturnURL: cfg.Yookassa.ReturnURL,
		HTTPClient: &http.Client{
			Timeout: cfg.Yookassa.RequestTimeout,
		},
	})
	if err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("create yookassa client: %w", err)
	}

	sessionsRepository := sessions.NewRepository(dbPool)
	productsRepository := productsrepo.NewRepository(dbPool)
	inventoryRepository := inventory.NewRepository(dbPool)
	cartRepository := cartrepo.NewRepository(dbPool)
	orderRepository := ordersrepo.NewRepository(dbPool)
	paymentRepository := paymentsrepo.NewRepository(dbPool)

	metricsRegistry := metricsinfra.NewRegistry()
	httpMetrics := metricsinfra.NewHTTPMetrics(metricsRegistry)
	metricsHandler := metricsinfra.NewHandler(metricsRegistry)
	jobMetrics := metricsinfra.NewJobsMetrics(metricsRegistry)
	paymentsMetrics := metricsinfra.NewPaymentMetrics(metricsRegistry)

	paymentProvider := metricsinfra.NewPaymentProviderDecorator(
		yookassa.NewProvider(yookassaClient),
		paymentsMetrics,
	)

	authService := authapp.NewService(usersService, tokenManager, sessionsRepository, cfg.JWT.RefreshTTL)
	productsService := productsapp.NewService(productsRepository, inventoryRepository, txManager)
	cartService := cartapp.NewService(cartRepository, txManager, productsService)
	orderService := ordersapp.NewService(
		orderRepository, inventoryRepository, cartService, paymentRepository, txManager, cfg.Orders.PaymentTTL,
	)
	paymentService := paymentsapp.NewService(
		paymentRepository, orderRepository, paymentProvider, inventoryRepository, txManager,
	)

	authHandler := authhttp.NewHandler(resp, usersService, authService, cookieManager)
	productsHandler := productshttp.NewHandler(resp, productsService)
	cartHandler := carthttp.NewHandler(resp, cartService)
	orderHandler := ordershttp.NewHandler(resp, orderService)
	paymentHandler := paymentshttp.NewHandler(resp, paymentService)

	webhookHandler := webhooks.NewHandler(log, resp, paymentService, paymentProvider)

	workers := worker.New(log, jobMetrics)
	jobs.RegisterOrderExpiration(workers, orderService, cfg.Orders, log)

	router := router.NewRouter(log, resp, tokenManager, router.RouterDependencies{
		AuthHandler:            authHandler,
		ProductsHandler:        productsHandler,
		CartHandler:            cartHandler,
		HealthHandler:          healthHandler,
		OrderHandler:           orderHandler,
		PaymentHandler:         paymentHandler,
		WebhookHandler:         webhookHandler,
		MetricsHandler:         metricsHandler,
		RequestMetricsRecorder: httpMetrics,
	})

	srv := server.New(cfg, log, router)
	return &App{server: srv, workers: workers, dbPool: dbPool}, nil
}

func (a *App) Run(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	a.workers.StartAll(runCtx)

	err := a.server.Run(runCtx)

	cancel()
	a.workers.Wait()

	if err != nil {
		return fmt.Errorf("run HTTP server: %w", err)
	}

	return nil
}

func (a *App) Close() {
	a.dbPool.Close()
}
