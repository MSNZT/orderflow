package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
}

type TxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

type txKey struct{}

func (m *TxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	const op = "postgres.TxManager.WithinTx"

	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("%s: begin tx: %w", op, err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txCtx := context.WithValue(ctx, txKey{}, tx)

	if err := fn(txCtx); err != nil {
		return fmt.Errorf("%s: execute tx: %w", op, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%s: commit tx: %w", op, err)
	}

	return nil
}

func ExecutorFromContext(ctx context.Context, defaultDB DBTX) DBTX {
	if tx, ok := ctx.Value(txKey{}).(DBTX); ok {
		return tx
	}
	return defaultDB
}
