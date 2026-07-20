package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/MSNZT/orderflow/internal/bootstrap"
	"github.com/MSNZT/orderflow/internal/config"
	"github.com/MSNZT/orderflow/internal/infrastructure/logger"
)

func main() {
	log := logger.New()

	if err := run(log); err != nil {
		log.Error(
			"application failed",
			logger.Err(err),
		)

		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	app, err := bootstrap.New(ctx, cfg, log)
	if err != nil {
		return fmt.Errorf("bootstrap application: %w", err)
	}
	defer app.Close()

	if err := app.Run(ctx); err != nil {
		return fmt.Errorf("run application: %w", err)
	}

	return nil
}
