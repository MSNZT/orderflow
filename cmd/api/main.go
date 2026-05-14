package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/MSNZT/orderflow/internal/config"
	"github.com/MSNZT/orderflow/internal/httpserver"
	"github.com/MSNZT/orderflow/internal/logger"
)

func main() {
	log := logger.New()

	cfg, err := config.Load()
	if err != nil {
		log.Error("cannot initialize config file", slog.String("error", err.Error()))
		os.Exit(1)
	}
	ctx := context.Background()

	server := httpserver.New(&cfg, log)

	if err := server.Run(ctx); err != nil {
		log.Error("application failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

}
