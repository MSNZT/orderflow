package payments

import (
	"context"
	"errors"
	"fmt"

	paymentsapp "github.com/MSNZT/orderflow/internal/app/payments"
	"github.com/MSNZT/orderflow/internal/infrastructure/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Repository struct {
	db postgres.DBTX
}

var _ paymentsapp.Repository = (*Repository)(nil)

func NewRepository(db postgres.DBTX) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, payment paymentsapp.Payment) (*paymentsapp.Payment, error) {
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

	var p paymentsapp.Payment

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
				return nil, fmt.Errorf("%s: %w", op, paymentsapp.ErrActivePaymentAlreadyExists)
			case pgErr.ConstraintName == "ux_payments_succeeded_order":
				return nil, fmt.Errorf("%s: %w", op, paymentsapp.ErrSucceededPaymentAlreadyExists)
			}
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &p, nil
}

func (r *Repository) GetActiveByOrderID(ctx context.Context, orderID uuid.UUID) (*paymentsapp.Payment, error) {
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

	var p paymentsapp.Payment

	err := db.QueryRow(ctx, query, orderID).Scan(
		&p.ID, &p.OrderID, &p.ProviderPaymentID, &p.IdempotencyKey, &p.Status, &p.AmountCents,
		&p.Currency, &p.ConfirmationURL, &p.Test, &p.CancellationParty, &p.CancellationReason,
		&p.ProviderCreatedAt, &p.SucceededAt, &p.CanceledAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, paymentsapp.ErrPaymentNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &p, nil
}

func (r *Repository) GetByProviderPaymentID(ctx context.Context, providerPaymentID string) (*paymentsapp.Payment, error) {
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

	var p paymentsapp.Payment

	err := db.QueryRow(ctx, query, providerPaymentID).Scan(
		&p.ID, &p.OrderID, &p.ProviderPaymentID, &p.IdempotencyKey, &p.Status, &p.AmountCents,
		&p.Currency, &p.ConfirmationURL, &p.Test, &p.CancellationParty, &p.CancellationReason,
		&p.ProviderCreatedAt, &p.SucceededAt, &p.CanceledAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, paymentsapp.ErrPaymentNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &p, nil
}

func (r *Repository) ApplyProviderCreateResult(
	ctx context.Context, paymentID uuid.UUID, result *paymentsapp.ProviderCreateResult) (*paymentsapp.Payment, error) {
	const op = "payments.repository.ApplyProviderCreateResult"

	query := `
		UPDATE payments
		SET provider_payment_id = $2,
			status = $3,
			confirmation_url = $4,
			test = $5,
			provider_created_at = $6,
			updated_at = now()
		WHERE id = $1 AND status = 'creating'
		RETURNING id, 
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

	var p paymentsapp.Payment
	if err := db.QueryRow(ctx, query, paymentID, result.ProviderPaymentID, result.Status, result.ConfirmationURL,
		result.Test, result.ProviderCreatedAt).Scan(
		&p.ID, &p.OrderID, &p.ProviderPaymentID, &p.IdempotencyKey, &p.Status, &p.AmountCents,
		&p.Currency, &p.ConfirmationURL, &p.Test, &p.CancellationParty, &p.CancellationReason,
		&p.ProviderCreatedAt, &p.SucceededAt, &p.CanceledAt, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, paymentsapp.ErrPaymentNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &p, nil
}

func (r *Repository) MarkFailed(ctx context.Context, paymentID uuid.UUID) error {
	const op = "payments.repository.MarkFailed"

	query := `
		UPDATE payments
		SET status = 'failed',
			updated_at = now()
		WHERE id = $1 AND status = 'creating'
		RETURNING status;
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	var resultStatus string
	if err := db.QueryRow(ctx, query, paymentID).Scan(&resultStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%s: %w", op, paymentsapp.ErrPaymentNotFound)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	if resultStatus != "failed" {
		return fmt.Errorf("%s: %w", op, paymentsapp.ErrPaymentStateConflict)
	}

	return nil
}
