package payments

import (
	"context"
	"errors"
	"fmt"

	"github.com/MSNZT/orderflow/internal/infrastructure/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Repository struct {
	db postgres.DBTX
}

func NewRepository(db postgres.DBTX) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, payment Payment) (*Payment, error) {
	const op = "payments.repository.Create"

	query := `
		INSERT INTO payments(
			id, order_id, idempotency_key, status, amount_cents, currency
		) VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING 
			id, 
			order_id, 
			provider_payment_id, 
			idempotency_key, 
			status, 
			amount_cents,
			currency, 
			confirmation_url, 
			test, 
			cancellation_party, 
			cancellation_reason, 
			provider_created_at, 
			succeeded_at, 
			canceled_at,
			created_at,
			updated_at;
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	var p Payment

	err := db.QueryRow(ctx, query, payment.ID, payment.OrderID, payment.IdempotencyKey, payment.Status,
		payment.AmountCents, payment.Currency).Scan(
		&p.ID, &p.OrderID, &p.ProviderPaymentID, &p.IdempotencyKey, &p.Status, &p.AmountCents,
		&p.Currency, &p.ConfirmationURL, &p.Test, &p.CancellationParty, &p.CancellationReason,
		&p.ProviderCreatedAt, &p.SucceededAt, &p.CanceledAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch {
			case pgErr.ConstraintName == "ux_payments_active_order":
				return nil, fmt.Errorf("%s: %w", op, ErrActivePaymentAlreadyExists)
			case pgErr.ConstraintName == "ux_payments_succeeded_order":
				return nil, fmt.Errorf("%s: %w", op, ErrSucceededPaymentAlreadyExists)
			}
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &p, nil
}

func (r *Repository) GetActiveByOrderID(ctx context.Context, orderID uuid.UUID) (*Payment, error) {
	const op = "payments.repository.GetActiveByOrderID"

	query := `
		SELECT 
			id, 
			order_id, 
			provider_payment_id, 
			idempotency_key, 
			status, 
			amount_cents,
			currency, 
			confirmation_url, 
			test, 
			cancellation_party, 
			cancellation_reason, 
			provider_created_at, 
			succeeded_at, 
			canceled_at,
			created_at,
			updated_at
		FROM payments
		WHERE order_id = $1
		AND status IN ('creating', 'pending', 'waiting_for_capture');
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	var p Payment

	err := db.QueryRow(ctx, query, orderID).Scan(
		&p.ID, &p.OrderID, &p.ProviderPaymentID, &p.IdempotencyKey, &p.Status, &p.AmountCents,
		&p.Currency, &p.ConfirmationURL, &p.Test, &p.CancellationParty, &p.CancellationReason,
		&p.ProviderCreatedAt, &p.SucceededAt, &p.CanceledAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, ErrPaymentNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &p, nil
}

func (r *Repository) GetByProviderPaymentID(ctx context.Context, providerPaymentID string) (*Payment, error) {
	const op = "payments.repository.GetByProviderPaymentID"

	query := `
		SELECT 
			id, 
			order_id, 
			provider_payment_id, 
			idempotency_key, 
			status, 
			amount_cents,
			currency, 
			confirmation_url, 
			test, 
			cancellation_party, 
			cancellation_reason, 
			provider_created_at, 
			succeeded_at, 
			canceled_at,
			created_at,
			updated_at
		FROM payments
		WHERE provider_payment_id = $1
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	var p Payment

	err := db.QueryRow(ctx, query, providerPaymentID).Scan(
		&p.ID, &p.OrderID, &p.ProviderPaymentID, &p.IdempotencyKey, &p.Status, &p.AmountCents,
		&p.Currency, &p.ConfirmationURL, &p.Test, &p.CancellationParty, &p.CancellationReason,
		&p.ProviderCreatedAt, &p.SucceededAt, &p.CanceledAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, ErrPaymentNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &p, nil
}
