package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/MSNZT/orderflow/internal/config"
	"github.com/MSNZT/orderflow/internal/httpserver"
	"github.com/MSNZT/orderflow/internal/logger"
	postgres "github.com/MSNZT/orderflow/internal/platform"
)

func main() {
	log := logger.New()

	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}
	ctx := context.Background()

	dbPool, err := postgres.NewPool(ctx, &cfg.DB)
	if err != nil {
		log.Error("failed to init pool", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer dbPool.Close()

	server := httpserver.New(&cfg, dbPool, log)

	if err := server.Run(ctx); err != nil {
		log.Error("application failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

}
