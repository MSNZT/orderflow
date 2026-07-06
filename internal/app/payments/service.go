package payments

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MSNZT/orderflow/internal/app/orders"
	"github.com/google/uuid"
)

type Service struct {
	repo            Repository
	ordersProvider  OrdersProvider
	paymentProvider PaymentProvider
}

func NewService(repo Repository, ordersProvider OrdersProvider, paymentProvider PaymentProvider) *Service {
	return &Service{repo: repo, ordersProvider: ordersProvider, paymentProvider: paymentProvider}
}

func (s *Service) CreatePayment(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (*Payment, error) {
	const op = "payments.service.CreatePayment"

	if userID == uuid.Nil {
		return nil, fmt.Errorf("%s: %w", op, ErrUserIDIsNil)
	}

	if orderID == uuid.Nil {
		return nil, fmt.Errorf("%s: %w", op, ErrOrderIDIsNil)
	}

	details, err := s.ordersProvider.GetDetailsByIDAndUserID(ctx, userID, orderID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if details == nil {
		return nil, fmt.Errorf(
			"%s: orders provider returned nil details",
			op,
		)
	}

	if err := s.validatePayableOrder(details); err != nil {
		return nil, err
	}

	activePayment, err := s.repo.GetActiveByOrderID(ctx, orderID)
	if err == nil {
		return s.processActivePayment(ctx, activePayment)
	}

	if !errors.Is(err, ErrPaymentNotFound) {
		return nil, fmt.Errorf("%s: get active payment: %w", op, err)
	}

	payment, err := s.createLocalPayment(ctx, details)
	if err != nil {
		return nil, err
	}

	return s.processActivePayment(ctx, payment)
}

func (s *Service) processActivePayment(
	ctx context.Context,
	payment *Payment,
) (*Payment, error) {
	const op = "payments.service.processActivePayment"

	if payment == nil {
		return nil, fmt.Errorf("%s: nil payment", op)
	}

	if payment.Status == StatusPending || payment.Status == StatusWaitingForCapture {
		return payment, nil
	}

	if payment.Status != StatusCreating {
		return nil, fmt.Errorf(
			"%s: payment %s has unexpected status %q: %w",
			op,
			payment.ID,
			payment.Status,
			ErrPaymentStateConflict,
		)
	}

	params := ProviderCreateParams{
		AmountCents:    payment.AmountCents,
		Currency:       payment.Currency,
		Description:    fmt.Sprintf("Payment for order %s", payment.OrderID),
		OrderID:        payment.OrderID,
		LocalPaymentID: payment.ID,
		IdempotencyKey: payment.IdempotencyKey,
	}

	result, providerErr := s.paymentProvider.CreatePayment(ctx, params)
	if providerErr != nil {
		if errors.Is(providerErr, ErrProviderRejected) {
			markErr := s.repo.MarkFailed(ctx, payment.ID)
			if markErr != nil {
				return nil, errors.Join(
					fmt.Errorf(
						"%s: provider rejected payment: %w",
						op,
						providerErr,
					),
					fmt.Errorf(
						"%s: mark payment failed: %w",
						op,
						markErr,
					),
				)
			}
		}

		return nil, fmt.Errorf(
			"%s: create provider payment: %w",
			op,
			providerErr,
		)
	}

	if result == nil {
		return nil, fmt.Errorf(
			"%s: nil provider result: %w",
			op,
			ErrProviderFailure,
		)
	}

	updatedPayment, err := s.repo.ApplyProviderCreateResult(
		ctx,
		payment.ID,
		result,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%s: apply provider result: %w",
			op,
			err,
		)
	}

	return updatedPayment, nil
}

func (s *Service) validatePayableOrder(details *orders.OrderDetails) error {
	const op = "payments.service.validatePayableOrder"

	if details.Status != orders.StatusPending {
		return fmt.Errorf("%s: %w", op, ErrOrderNotPayable)
	}

	if !details.ExpiresAt.After(time.Now()) {
		return fmt.Errorf("%s: %w", op, ErrOrderExpired)
	}

	if details.TotalPriceCents <= 0 {
		return fmt.Errorf("%s: order has invalid amount: %d", op, details.TotalPriceCents)
	}

	currency := strings.TrimSpace(details.Currency)
	if currency == "" {
		return fmt.Errorf("%s: order has empty currency: %q", op, details.Currency)
	}

	return nil
}

func (s *Service) createLocalPayment(ctx context.Context, details *orders.OrderDetails) (*Payment, error) {
	const op = "payments.service.createLocalPayment"

	paymentID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("%s: generate payment ID: %w", op, err)
	}

	newPayment := Payment{
		ID:             paymentID,
		OrderID:        details.ID,
		IdempotencyKey: uuid.New(),
		Status:         StatusCreating,
		AmountCents:    details.TotalPriceCents,
		Currency:       strings.TrimSpace(details.Currency),
	}

	payment, err := s.repo.Create(ctx, newPayment)
	if err != nil {
		if !errors.Is(err, ErrActivePaymentAlreadyExists) {
			return nil, fmt.Errorf(
				"%s: create local payment: %w",
				op,
				err,
			)
		}

		payment, err = s.repo.GetActiveByOrderID(ctx, details.ID)
		if err != nil {
			return nil, fmt.Errorf(
				"%s: get concurrently created payment: %w",
				op,
				err,
			)
		}
	}

	return payment, nil
}
