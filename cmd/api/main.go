package main

import (
	"context"
	"os"

	"github.com/MSNZT/orderflow/internal/config"
	"github.com/MSNZT/orderflow/internal/httpserver"
	"github.com/MSNZT/orderflow/internal/logger"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	log := logger.New()
	server := httpserver.New(&cfg, log)

	if err := server.Run(ctx); err != nil {
		log.Error("Application failed", "error", err)
		os.Exit(1)
	}

}
