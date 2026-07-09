package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/MSNZT/orderflow/internal/config"
	"github.com/MSNZT/orderflow/internal/platform/worker"
)

type OrderProcess interface {
	ExpireOverdueOrders(ctx context.Context, now time.Time, limit int) (int, error)
}

func RegisterOrderExpiration(
	workers *worker.Manager, orderService OrderProcess, cfg config.OrdersConfig, log *slog.Logger) {
	runFunc := func(ctx context.Context) error {
		count, err := orderService.ExpireOverdueOrders(ctx, time.Now(), int(cfg.ExpireBatchLimit))
		if err != nil {
			return err
		}

		if count > 0 {
			log.Info("expired overdue orders", "count", count)
		}

		return nil
	}
	workers.RegisterPeriodic("order-expiration", cfg.ExpireInterval, false, worker.JobFunc(runFunc))
}
