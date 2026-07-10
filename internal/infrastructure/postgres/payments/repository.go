package payments

import (
	"context"
	"errors"
	"fmt"
	"time"

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

func (r *Repository) CancelActiveByOrderID(ctx context.Context, orderID uuid.UUID, now time.Time) error {
	const op = "payments.repository.CancelActiveByOrderID"

	query := `
		UPDATE payments
		SET status = 'canceled',
			updated_at = $2,
			canceled_at = $2
		WHERE order_id = $1 
			AND status IN ('creating', 'pending');
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	_, err := db.Exec(ctx, query, orderID, now)
	if err != nil {
		return fmt.Errorf("%s: failed to cancel active payment %w", op, err)
	}

	return nil
}

func (r *Repository) HasWaitingForCaptureByOrderID(ctx context.Context, orderID uuid.UUID) (bool, error) {
	const op = "payments.repository.HasWaitingForCaptureByOrderID"

	if orderID == uuid.Nil {
		return false, fmt.Errorf("%s: %w", op, paymentsapp.ErrOrderIDIsNil)
	}

	query := `
		SELECT EXISTS (
			SELECT 1
			FROM payments
			WHERE order_id = $1
			  AND status = 'waiting_for_capture'
		)
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	var exists bool
	if err := db.QueryRow(ctx, query, orderID).Scan(&exists); err != nil {
		return false, fmt.Errorf(
			"%s: check waiting for capture payment: %w",
			op,
			err,
		)
	}

	return exists, nil
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
			payment, err := r.resolveApplyResultNoUpdate(ctx, paymentID, result.ProviderPaymentID)

			if err != nil {
				return nil, fmt.Errorf("%s: %w", op, err)
			}

			return payment, nil
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &p, nil
}

func (r *Repository) MarkFailed(ctx context.Context, paymentID uuid.UUID) error {
	const op = "payments.repository.MarkFailed"

	if paymentID == uuid.Nil {
		return fmt.Errorf("%s: %w", op, paymentsapp.ErrPaymentIDIsNil)
	}

	query := `
		WITH existing AS (
			SELECT id
			FROM payments
			WHERE id = $1
		),
		updated AS (
			UPDATE payments
			SET status = $2,
				updated_at = now()
			WHERE id = $1
			  AND status = $3
			RETURNING id
		)
		SELECT
			EXISTS (SELECT 1 FROM existing) AS payment_existing,
			EXISTS (SELECT 1 FROM updated) AS payment_updated
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	var paymentExisting, paymentUpdated bool
	if err := db.QueryRow(
		ctx,
		query,
		paymentID,
		string(paymentsapp.StatusFailed),
		string(paymentsapp.StatusCreating),
	).Scan(&paymentExisting, &paymentUpdated); err != nil {
		return fmt.Errorf("%s: failed to mark payment failed: %w", op, err)
	}

	if !paymentExisting {
		return fmt.Errorf("%s: %w", op, paymentsapp.ErrPaymentNotFound)
	}

	if !paymentUpdated {
		return fmt.Errorf("%s: %w", op, paymentsapp.ErrPaymentStatusTransitionInvalid)
	}

	return nil
}

func (r *Repository) MarkSucceeded(ctx context.Context, paymentID uuid.UUID) error {
	const op = "payments.repository.MarkSucceeded"

	if paymentID == uuid.Nil {
		return fmt.Errorf("%s: %w", op, paymentsapp.ErrPaymentIDIsNil)
	}

	query := `
		WITH existing AS (
			SELECT id
			FROM payments
			WHERE id = $1
		),
		updated AS (
			UPDATE payments
			SET status = $2,
				succeeded_at = now(),
				updated_at = now()
			WHERE id = $1
			  AND status IN ($3, $4)
			RETURNING id
		)
		SELECT
			EXISTS (SELECT 1 FROM existing) AS payment_existing,
			EXISTS (SELECT 1 FROM updated) AS payment_updated
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	var paymentExisting, paymentUpdated bool
	if err := db.QueryRow(
		ctx,
		query,
		paymentID,
		string(paymentsapp.StatusSucceeded),
		string(paymentsapp.StatusPending),
		string(paymentsapp.StatusWaitingForCapture),
	).Scan(&paymentExisting, &paymentUpdated); err != nil {
		return fmt.Errorf("%s: failed to mark payment succeeded: %w", op, err)
	}

	if !paymentExisting {
		return fmt.Errorf("%s: %w", op, paymentsapp.ErrPaymentNotFound)
	}

	if !paymentUpdated {
		return fmt.Errorf("%s: %w", op, paymentsapp.ErrPaymentStatusTransitionInvalid)
	}

	return nil
}

func (r *Repository) MarkWaitingForCapture(ctx context.Context, paymentID uuid.UUID) error {
	const op = "payments.repository.MarkWaitingForCapture"

	if paymentID == uuid.Nil {
		return fmt.Errorf("%s: %w", op, paymentsapp.ErrPaymentIDIsNil)
	}

	query := `
		WITH existing AS (
			SELECT id
			FROM payments
			WHERE id = $1
		),
		updated AS (
			UPDATE payments
			SET status = $2,
				updated_at = now()
			WHERE id = $1
			  AND status IN ($3,$4)
			RETURNING id
		)
		SELECT
			EXISTS (SELECT 1 FROM existing) AS payment_existing,
			EXISTS (SELECT 1 FROM updated) AS payment_updated
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	var paymentExisting, paymentUpdated bool
	if err := db.QueryRow(
		ctx,
		query,
		paymentID,
		string(paymentsapp.StatusWaitingForCapture),
		string(paymentsapp.StatusCreating),
		string(paymentsapp.StatusPending),
	).Scan(&paymentExisting, &paymentUpdated); err != nil {
		return fmt.Errorf("%s: failed to mark payment waiting for capture: %w", op, err)
	}

	if !paymentExisting {
		return fmt.Errorf("%s: %w", op, paymentsapp.ErrPaymentNotFound)
	}

	if !paymentUpdated {
		return fmt.Errorf("%s: %w", op, paymentsapp.ErrPaymentStatusTransitionInvalid)
	}

	return nil
}

func (r *Repository) MarkCanceled(ctx context.Context, paymentID uuid.UUID) error {
	const op = "payments.repository.MarkCanceled"

	if paymentID == uuid.Nil {
		return fmt.Errorf("%s: %w", op, paymentsapp.ErrPaymentIDIsNil)
	}

	query := `
		WITH existing AS (
			SELECT id
			FROM payments
			WHERE id = $1
		),
		updated AS (
			UPDATE payments
			SET status = $2,
				canceled_at = now(),
				updated_at = now()
			WHERE id = $1
			  AND status IN ($3, $4)
			RETURNING id
		)
		SELECT
			EXISTS (SELECT 1 FROM existing) AS payment_existing,
			EXISTS (SELECT 1 FROM updated) AS payment_updated
	`

	db := postgres.ExecutorFromContext(ctx, r.db)

	var paymentExisting, paymentUpdated bool
	if err := db.QueryRow(
		ctx,
		query,
		paymentID,
		string(paymentsapp.StatusCanceled),
		string(paymentsapp.StatusPending),
		string(paymentsapp.StatusWaitingForCapture),
	).Scan(&paymentExisting, &paymentUpdated); err != nil {
		return fmt.Errorf("%s: failed to mark payment canceled: %w", op, err)
	}

	if !paymentExisting {
		return fmt.Errorf("%s: %w", op, paymentsapp.ErrPaymentNotFound)
	}

	if !paymentUpdated {
		return fmt.Errorf("%s: %w", op, paymentsapp.ErrPaymentStatusTransitionInvalid)
	}

	return nil
}

func (r *Repository) resolveApplyResultNoUpdate(
	ctx context.Context,
	paymentID uuid.UUID,
	providerPaymentID string,
) (*paymentsapp.Payment, error) {
	const op = "payments.repository.resolveApplyResultNoUpdate"

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
		WHERE id = $1
	`

	db := postgres.ExecutorFromContext(ctx, r.db)
	var p paymentsapp.Payment
	if err := db.QueryRow(ctx, query, paymentID).Scan(
		&p.ID, &p.OrderID, &p.ProviderPaymentID, &p.IdempotencyKey, &p.Status, &p.AmountCents,
		&p.Currency, &p.ConfirmationURL, &p.Test, &p.CancellationParty, &p.CancellationReason,
		&p.ProviderCreatedAt, &p.SucceededAt, &p.CanceledAt, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, paymentsapp.ErrPaymentNotFound)
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if p.ProviderPaymentID == nil {
		return nil, fmt.Errorf(
			"%s: %s has no provider payment ID: %w",
			op,
			paymentID,
			paymentsapp.ErrPaymentStateConflict,
		)
	}

	if *p.ProviderPaymentID != providerPaymentID {
		return nil, fmt.Errorf(
			"%s: provider payment ID mismatch: expected %q, got %q: %w",
			op,
			providerPaymentID,
			*p.ProviderPaymentID,
			paymentsapp.ErrPaymentStateConflict,
		)
	}

	return &p, nil
}
