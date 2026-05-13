package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/MSNZT/orderflow/internal/config"
	"github.com/MSNZT/orderflow/internal/router"
)

type App struct {
	server *http.Server
	config *config.Config
}

func New(config *config.Config, log *slog.Logger) *App {
	router := router.NewRouter()

	return &App{
		server: &http.Server{
			Addr:         config.HTTPServer.Addr,
			Handler:      router,
			ReadTimeout:  config.HTTPServer.Timeout,
			WriteTimeout: config.HTTPServer.Timeout,
			IdleTimeout:  config.HTTPServer.IdleTimeout,
		},
		config: config,
	}
}

func (a *App) Run(ctx context.Context, log *slog.Logger) error {
	stop := make(chan os.Signal, 1)
	serverErrors := make(chan error, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Info("Starting server on", "addr", a.server.Addr)

		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	select {
	case err := <-serverErrors:
		log.Error("Server error", "error", err)
		return fmt.Errorf("Server error: %w", err)
	case <-stop:
		log.Info("Termination signal received. Stopping server...")
	}

	ctx, cancel := context.WithTimeout(ctx, a.config.HTTPServer.ShutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		log.Error("server shutdown failed:", "err", err)
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Info("Server successfully stopped")
	return nil
}
