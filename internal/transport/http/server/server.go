package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/MSNZT/orderflow/internal/config"
)

type Server struct {
	server *http.Server
	config *config.Config
	logger *slog.Logger
}

func New(config *config.Config, log *slog.Logger, handler http.Handler) *Server {
	return &Server{
		server: &http.Server{
			Addr:              config.HTTP.Addr,
			Handler:           handler,
			ReadTimeout:       config.HTTP.ReadTimeout,
			ReadHeaderTimeout: config.HTTP.ReadHeaderTimeout,
			WriteTimeout:      config.HTTP.WriteTimeout,
			IdleTimeout:       config.HTTP.IdleTimeout,
		},
		config: config,
		logger: log,
	}
}

func (s *Server) Run(ctx context.Context) error {
	serverErrors := make(chan error, 1)

	go func() {
		s.logger.Info("http server started", slog.String("addr", s.server.Addr))

		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	select {
	case <-ctx.Done():
		s.logger.Info("shutdown started", slog.String("reason", "context canceled"))
	case err := <-serverErrors:
		s.logger.Error("server error", slog.String("error", err.Error()))
		return fmt.Errorf("server error: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.config.HTTP.ShutdownTimeout)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("http server shutdown failed", slog.String("error", err.Error()))

		if err := s.server.Close(); err != nil {
			return fmt.Errorf("server close: %w", err)
		}

		return fmt.Errorf("server shutdown: %w", err)
	}

	s.logger.Info("shutdown completed")
	return nil
}
