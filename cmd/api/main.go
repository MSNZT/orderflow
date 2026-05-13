package main

import (
	"context"
	"os"

	"github.com/MSNZT/orderflow/internal/config"
	"github.com/MSNZT/orderflow/internal/httpserver"
	"github.com/MSNZT/orderflow/internal/router/middleware"
)

func main() {
	cfg := config.MustLoad()
	ctx := context.Background()

	log := middleware.SetopLogger()
	server := httpserver.New(&cfg, log)

	if err := server.Run(ctx, log); err != nil {
		log.Error("Application failed", "error", err)
		os.Exit(1)
	}

}
