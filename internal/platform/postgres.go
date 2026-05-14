package postgres

import (
	"context"
	"fmt"

	"github.com/MSNZT/orderflow/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, config *config.DBConfig) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dsn: %w", err)
	}

	cfg.MaxConns = config.MaxOpenConns
	cfg.MinConns = config.MaxIddleConns
	cfg.MaxConnLifetime = config.ConnMaxLifetime

	conn, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping postgres server: %w", err)
	}

	return conn, nil
}
