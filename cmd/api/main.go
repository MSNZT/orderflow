package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/MSNZT/orderflow/internal/auth"
	"github.com/MSNZT/orderflow/internal/cart"
	"github.com/MSNZT/orderflow/internal/config"
	"github.com/MSNZT/orderflow/internal/health"
	"github.com/MSNZT/orderflow/internal/httpserver"
	"github.com/MSNZT/orderflow/internal/inventory"
	"github.com/MSNZT/orderflow/internal/logger"
	"github.com/MSNZT/orderflow/internal/platform/postgres"
	"github.com/MSNZT/orderflow/internal/products"
	"github.com/MSNZT/orderflow/internal/router"
	"github.com/MSNZT/orderflow/internal/sessions"
	"github.com/MSNZT/orderflow/internal/token"
	"github.com/MSNZT/orderflow/internal/users"
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

	usersRepository := users.NewRepository(dbPool)
	hasher := users.NewBcryptHasher(cost)
	usersService := users.NewService(usersRepository, hasher)
	tokenManager := token.NewManager(cfg.JWT.Secret, cfg.JWT.AccessTTL)

	sessionsRepository := sessions.NewRepository(dbPool)
	authService := auth.NewService(usersService, tokenManager, sessionsRepository, cfg.JWT.RefreshTTL)
	authHandler := auth.NewHandler(log, usersService, authService)

	productsRepository := products.NewRepository(dbPool)
	inventoryRepository := inventory.NewRepository(dbPool)
	productsService := products.NewService(productsRepository, inventoryRepository, txManager)
	productsHandler := products.NewHandler(log, productsService)

	cartRepository := cart.NewRepository(dbPool)
	cartService := cart.NewService(cartRepository, txManager)
	cartHandler := cart.NewHandler(log, cartService)

	router := router.NewRouter(log, tokenManager, router.RouterDependencies{
		AuthHandler:     authHandler,
		ProductsHandler: productsHandler,
		CartHandler:     cartHandler,
		HealthHandler:   healthHandler,
	})

	server := httpserver.New(cfg, log, router)

	if err := server.Run(ctx); err != nil {
		log.Error("application failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

}
