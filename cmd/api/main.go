package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	authapp "github.com/MSNZT/orderflow/internal/app/auth"
	cartapp "github.com/MSNZT/orderflow/internal/app/cart"
	ordersapp "github.com/MSNZT/orderflow/internal/app/orders"
	productsapp "github.com/MSNZT/orderflow/internal/app/products"
	usersapp "github.com/MSNZT/orderflow/internal/app/users"
	"github.com/MSNZT/orderflow/internal/config"
	"github.com/MSNZT/orderflow/internal/infrastructure/logger"
	"github.com/MSNZT/orderflow/internal/infrastructure/password"
	"github.com/MSNZT/orderflow/internal/infrastructure/postgres"
	cartrepo "github.com/MSNZT/orderflow/internal/infrastructure/postgres/cart"
	"github.com/MSNZT/orderflow/internal/infrastructure/postgres/inventory"
	ordersrepo "github.com/MSNZT/orderflow/internal/infrastructure/postgres/orders"
	productsrepo "github.com/MSNZT/orderflow/internal/infrastructure/postgres/products"
	"github.com/MSNZT/orderflow/internal/infrastructure/postgres/sessions"
	usersrepo "github.com/MSNZT/orderflow/internal/infrastructure/postgres/users"
	"github.com/MSNZT/orderflow/internal/infrastructure/token"
	authhttp "github.com/MSNZT/orderflow/internal/transport/http/auth"
	carthttp "github.com/MSNZT/orderflow/internal/transport/http/cart"
	"github.com/MSNZT/orderflow/internal/transport/http/health"
	ordershttp "github.com/MSNZT/orderflow/internal/transport/http/orders"
	productshttp "github.com/MSNZT/orderflow/internal/transport/http/products"
	"github.com/MSNZT/orderflow/internal/transport/http/router"
	"github.com/MSNZT/orderflow/internal/transport/http/server"
)

func main() {
	log := logger.New()

	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbPool, err := postgres.NewPool(ctx, &cfg.Postgres)
	if err != nil {
		log.Error("failed to connect to postgres", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer dbPool.Close()

	txManager := postgres.NewTxManager(dbPool)

	const cost = 12
	healthHandler := health.NewHandler(log, dbPool)

	usersRepository := usersrepo.NewRepository(dbPool)
	hasher := password.NewBcryptHasher(cost)
	usersService := usersapp.NewService(usersRepository, hasher)
	tokenManager := token.NewManager(cfg.JWT.Secret, cfg.JWT.AccessTTL)

	sessionsRepository := sessions.NewRepository(dbPool)
	authService := authapp.NewService(usersService, tokenManager, sessionsRepository, cfg.JWT.RefreshTTL)
	authHandler := authhttp.NewHandler(log, usersService, authService)

	productsRepository := productsrepo.NewRepository(dbPool)
	inventoryRepository := inventory.NewRepository(dbPool)
	productsService := productsapp.NewService(productsRepository, inventoryRepository, txManager)
	productsHandler := productshttp.NewHandler(log, productsService)

	cartRepository := cartrepo.NewRepository(dbPool)
	cartService := cartapp.NewService(cartRepository, txManager, productsService)
	cartHandler := carthttp.NewHandler(log, cartService)

	orderRepository := ordersrepo.NewRepository(dbPool)
	orderService := ordersapp.NewService(orderRepository, inventoryRepository, cartService, txManager)
	orderHandler := ordershttp.NewHandler(log, orderService)

	router := router.NewRouter(log, tokenManager, router.RouterDependencies{
		AuthHandler:     authHandler,
		ProductsHandler: productsHandler,
		CartHandler:     cartHandler,
		HealthHandler:   healthHandler,
		OrderHandler:    orderHandler,
	})

	srv := server.New(cfg, log, router)

	if err := srv.Run(ctx); err != nil {
		log.Error("application failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

}
