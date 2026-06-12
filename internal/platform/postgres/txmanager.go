package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

func (m *TxManager) WithinTx(ctx context.Context, fn func(ctx context.Context, db DBTX) error) error {
	const op = "postgres.TxManager.WithinTx"

	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("%s: begin tx: %w", op, err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := fn(ctx, tx); err != nil {
		return fmt.Errorf("%s: execute tx: %w", op, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%s: commit tx: %w", op, err)
	}

	return nil
}
