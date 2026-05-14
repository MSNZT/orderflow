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
	logger *slog.Logger
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
		logger: log,
	}
}

func (a *App) Run(ctx context.Context) error {
	stop := make(chan os.Signal, 1)
	serverErrors := make(chan error, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		a.logger.Info("http server started", slog.String("addr", a.server.Addr))

		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	select {
	case err := <-serverErrors:
		a.logger.Error("server error", slog.String("error", err.Error()))
		return fmt.Errorf("server error: %w", err)
	case <-stop:
		a.logger.Info("Termination signal received. Stopping server...")
	}

	ctx, cancel := context.WithTimeout(ctx, a.config.HTTPServer.ShutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		a.logger.Error("http server shutdown failed:", slog.String("err", err.Error()))
		return fmt.Errorf("http server shutdown failed: %w", err)
	}

	a.logger.Info("Server successfully stopped")
	return nil
}
