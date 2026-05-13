package main

import (
	"context"

	"github.com/MSNZT/orderflow/internal/config"
	"github.com/MSNZT/orderflow/internal/httpserver"
)

func main() {
	config := config.MustLoad()
	ctx := context.Background()
	server := httpserver.New(&config)

	server.Run(ctx)
}
