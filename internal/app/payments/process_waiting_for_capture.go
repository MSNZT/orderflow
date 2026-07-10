package payments

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/MSNZT/orderflow/internal/app/inventory"
	"github.com/MSNZT/orderflow/internal/app/orders"
	"github.com/google/uuid"
)

type waitingDecision uint8

const (
	waitingDecisionUnknown waitingDecision = iota
	waitingDecisionCapture
	waitingDecisionCancelAndExpire
	waitingDecisionCancelOnly
)

type waitingForCapturePlan struct {
	decision          waitingDecision
	localPaymentID    uuid.UUID
	providerPaymentID string
	orderID           uuid.UUID
	amountCents       int64
	currency          string
	reservedItems     []inventory.ReservedItem
}

func (s *Service) ProcessWaitingForCapturePayment(
	ctx context.Context,
	providerPayment ProviderPayment,
	now time.Time,
) error {
	const op = "payments.service.ProcessWaitingForCapturePayment"

	if now.IsZero() {
		now = time.Now().UTC()
	}

	plan, err := s.prepareWaitingForCapture(ctx, providerPayment, now)
	if err != nil {
		return fmt.Errorf("%s: prepare action: %w", op, err)
	}

	switch plan.decision {
	case waitingDecisionCapture:
		if err := s.captureWaitingPayment(ctx, plan); err != nil {
			return fmt.Errorf("%s: capture payment: %w", op, err)
		}

	case waitingDecisionCancelAndExpire:
		if err := s.cancelWaitingPayment(ctx, plan, true); err != nil {
			return fmt.Errorf("%s: cancel and expire: %w", op, err)
		}

	case waitingDecisionCancelOnly:
		if err := s.cancelWaitingPayment(ctx, plan, false); err != nil {
			return fmt.Errorf("%s: cancel payment: %w", op, err)
		}

	default:
		return fmt.Errorf("%s: payment decision was not selected", op)
	}

	return nil
}

func (s *Service) prepareWaitingForCapture(
	ctx context.Context,
	providerPayment ProviderPayment,
	now time.Time,
) (waitingForCapturePlan, error) {
	const op = "payments.service.prepareWaitingForCapture"

	providerPaymentID := strings.TrimSpace(providerPayment.ID)
	if providerPaymentID == "" {
		return waitingForCapturePlan{}, fmt.Errorf(
			"%s: %w",
			op,
			ErrProviderPaymentIDRequired,
		)
	}

	var plan waitingForCapturePlan

	err := s.txManager.WithinTx(ctx, func(txCtx context.Context) error {
		payment, err := s.repo.GetByProviderPaymentID(
			txCtx,
			providerPaymentID,
		)
		if err != nil {
			return fmt.Errorf("get payment: %w", err)
		}

		if payment == nil {
			return fmt.Errorf("payment repository returned nil")
		}

		if err := validateProviderPayment(
			providerPayment,
			payment,
			StatusWaitingForCapture,
		); err != nil {
			return fmt.Errorf("validate provider payment: %w", err)
		}

		orderDetails, err := s.ordersProvider.GetDetailsByID(
			txCtx,
			payment.OrderID,
		)
		if err != nil {
			return fmt.Errorf("get order details: %w", err)
		}

		if orderDetails == nil {
			return fmt.Errorf("orders provider returned nil details")
		}

		switch payment.Status {
		case StatusCreating, StatusPending:
			if err := s.repo.MarkWaitingForCapture(
				txCtx,
				payment.ID,
			); err != nil {
				return fmt.Errorf(
					"mark payment waiting for capture: %w",
					err,
				)
			}

		case StatusWaitingForCapture:
			// Повторная обработка — нормальный сценарий.

		case StatusCanceled:
			// Разрешаем только provider cancel.
			// Решение ниже будет зависеть от заказа.

		case StatusSucceeded:
			return fmt.Errorf(
				"payment already succeeded: %w",
				ErrPaymentStatusTransitionInvalid,
			)

		default:
			return fmt.Errorf(
				"unexpected payment status %q: %w",
				payment.Status,
				ErrPaymentStatusTransitionInvalid,
			)
		}

		switch orderDetails.Status {
		case orders.StatusPending:
			if orderDetails.ExpiresAt.After(now) {
				if payment.Status == StatusCanceled {
					return fmt.Errorf(
						"canceled payment belongs to active order: %w",
						ErrPaymentStatusTransitionInvalid,
					)
				}

				plan.decision = waitingDecisionCapture
			} else {
				plan.decision = waitingDecisionCancelAndExpire
			}

		case orders.StatusExpired, orders.StatusCanceled:
			plan.decision = waitingDecisionCancelOnly

		case orders.StatusPaid:
			return fmt.Errorf(
				"paid order has waiting provider payment: %w",
				ErrPaymentStatusTransitionInvalid,
			)

		default:
			return fmt.Errorf(
				"unexpected order status %q",
				orderDetails.Status,
			)
		}

		plan.localPaymentID = payment.ID
		plan.providerPaymentID = providerPaymentID
		plan.orderID = orderDetails.ID
		plan.amountCents = payment.AmountCents
		plan.currency = payment.Currency
		plan.reservedItems = reservedItemsFromOrder(orderDetails)

		return nil
	})
	if err != nil {
		return waitingForCapturePlan{}, fmt.Errorf("%s: %w", op, err)
	}

	return plan, nil
}

func (s *Service) captureWaitingPayment(
	ctx context.Context,
	plan waitingForCapturePlan,
) error {
	result, err := s.paymentProvider.CapturePayment(
		ctx,
		CapturePaymentInput{
			ProviderPaymentID: plan.providerPaymentID,
			IdempotencyKey:    "capture:" + plan.localPaymentID.String(),
			AmountCents:       plan.amountCents,
			Currency:          plan.currency,
		},
	)
	if err != nil {
		return fmt.Errorf("provider capture: %w", err)
	}

	if result == nil {
		return fmt.Errorf("provider returned nil captured payment")
	}

	if result.ID != plan.providerPaymentID {
		return fmt.Errorf("captured payment id mismatch")
	}

	if result.Status != StatusSucceeded {
		return fmt.Errorf(
			"unexpected captured payment status %q",
			result.Status,
		)
	}

	if err := s.ProcessSucceededPayment(ctx, *result); err != nil {
		return fmt.Errorf("process succeeded payment: %w", err)
	}

	return nil
}

func (s *Service) cancelWaitingPayment(
	ctx context.Context,
	plan waitingForCapturePlan,
	expireOrder bool,
) error {
	result, err := s.paymentProvider.CancelPayment(
		ctx,
		CancelPaymentInput{
			ProviderPaymentID: plan.providerPaymentID,
			IdempotencyKey:    "cancel:" + plan.localPaymentID.String(),
		},
	)
	if err != nil {
		return fmt.Errorf("provider cancel: %w", err)
	}

	if result == nil {
		return fmt.Errorf("provider returned nil canceled payment")
	}

	if result.ID != plan.providerPaymentID {
		return fmt.Errorf("canceled payment id mismatch")
	}

	if result.Status != StatusCanceled {
		return fmt.Errorf(
			"unexpected canceled payment status %q",
			result.Status,
		)
	}

	if err := s.finalizeWaitingPaymentCancellation(
		ctx,
		plan,
		expireOrder,
	); err != nil {
		return fmt.Errorf("finalize cancellation: %w", err)
	}

	return nil
}

func (s *Service) finalizeWaitingPaymentCancellation(
	ctx context.Context,
	plan waitingForCapturePlan,
	expireOrder bool,
) error {
	return s.txManager.WithinTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.MarkCanceled(
			txCtx,
			plan.localPaymentID,
		); err != nil {
			return fmt.Errorf("mark payment canceled: %w", err)
		}

		if !expireOrder {
			return nil
		}

		if err := s.ordersProvider.MarkExpired(
			txCtx,
			plan.orderID,
		); err != nil {
			return fmt.Errorf("mark order expired: %w", err)
		}

		if err := s.inventoryProvider.ReleaseReservedQuantities(
			txCtx,
			plan.reservedItems,
		); err != nil {
			return fmt.Errorf("release reserved inventory: %w", err)
		}

		return nil
	})
}
