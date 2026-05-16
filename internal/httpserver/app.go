package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/MSNZT/orderflow/internal/config"
	"github.com/MSNZT/orderflow/internal/router"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	server *http.Server
	config *config.Config
	logger *slog.Logger
}

func New(config *config.Config, dbPool *pgxpool.Pool, log *slog.Logger) *App {
	router := router.NewRouter(dbPool, log)

	return &App{
		server: &http.Server{
			Addr:              config.HTTP.Addr,
			Handler:           router,
			ReadTimeout:       config.HTTP.ReadTimeout,
			ReadHeaderTimeout: config.HTTP.ReadHeaderTimeout,
			WriteTimeout:      config.HTTP.WriteTimeout,
			IdleTimeout:       config.HTTP.IdleTimeout,
		},
		config: config,
		logger: log,
	}
}

func (a *App) Run(ctx context.Context) error {
	serverErrors := make(chan error, 1)

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
		return fmt.Errorf("server error: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.config.HTTP.ShutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		a.logger.Error("http server shutdown failed", slog.String("error", err.Error()))

		if err := a.server.Close(); err != nil {
			return fmt.Errorf("server close: %w", err)
		}

		return fmt.Errorf("server shutdown: %w", err)
	}

	a.logger.Info("shutdown completed")
	return nil
}
