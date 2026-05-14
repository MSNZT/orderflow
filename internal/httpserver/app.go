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
			Addr:              config.HTTPServer.Addr,
			Handler:           router,
			ReadTimeout:       config.HTTPServer.ReadTimeout,
			ReadHeaderTimeout: config.HTTPServer.ReadHeaderTimeout,
			WriteTimeout:      config.HTTPServer.WriteTimeout,
			IdleTimeout:       config.HTTPServer.IdleTimeout,
		},
		config: config,
		logger: log,
	}
}

func (a *App) Run(ctx context.Context) error {
	stop := make(chan os.Signal, 1)
	serverErrors := make(chan error, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	go func() {
		a.logger.Info("http server started", slog.String("addr", a.server.Addr))

		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	select {
	case <-ctx.Done():
		a.logger.Info("shutdown started", slog.String("reason", "context canceled"))
	case err := <-serverErrors:
		a.logger.Error("server error", slog.String("error", err.Error()))
	case <-stop:
		a.logger.Info("shutdown started")
	}

	shutDown, cancel := context.WithTimeout(context.Background(), a.config.HTTPServer.ShutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(shutDown); err != nil {
		a.logger.Error("http server shutdown failed", slog.String("error", err.Error()))

		if err := a.server.Close(); err != nil {
			return fmt.Errorf("server close: %w", err)
		}

		return fmt.Errorf("server shutdown: %w", err)
	}

	a.logger.Info("shutdown completed")
	return nil
}
