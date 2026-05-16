package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/MSNZT/orderflow/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, config *config.PostgresConfig) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dsn: %w", err)
	}

	cfg.MaxConns = config.MaxConns
	cfg.MinConns = config.MinConns
	cfg.MaxConnLifetime = config.MaxConnLifetime

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	const maxAttempts = 5
	delay := 1 * time.Second

	var pingErr error
	for i := 0; i < maxAttempts; i++ {
		pingErr = pool.Ping(ctx)
		if pingErr == nil {
			return pool, nil
		}

		select {
		case <-ctx.Done():
			pool.Close()
			return nil, ctx.Err()
		case <-time.After(delay):
		}

		delay *= 2
	}

	if pingErr != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres server: %w", pingErr)
	}

	return pool, nil
}
