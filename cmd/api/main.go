package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/MSNZT/orderflow/internal/config"
	"github.com/MSNZT/orderflow/internal/httpserver"
	"github.com/MSNZT/orderflow/internal/logger"
	"github.com/MSNZT/orderflow/internal/platform/postgres"
)

func main() {
	log := logger.New()

	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}
	ctx := context.Background()

	dbPool, err := postgres.NewPool(ctx, &cfg.Postgres)
	if err != nil {
		log.Error("failed to connecet to postgres", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer dbPool.Close()

	server := httpserver.New(&cfg, dbPool, log)

	if err := server.Run(ctx); err != nil {
		log.Error("application failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

}
