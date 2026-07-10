package payments

import (
	"context"
	"fmt"
	"strings"

	"github.com/MSNZT/orderflow/internal/app/orders"
)

func (s *Service) ProcessCanceledPayment(ctx context.Context, providerPayment ProviderPayment) error {
	const op = "payments.service.ProcessCanceledPayment"

	providerPaymentID := strings.TrimSpace(providerPayment.ID)
	if providerPaymentID == "" {
		return fmt.Errorf("%s: %w", op, ErrProviderPaymentIDRequired)
	}

	err := s.txManager.WithinTx(ctx, func(txCtx context.Context) error {
		payment, err := s.repo.GetByProviderPaymentID(txCtx, providerPaymentID)
		if err != nil {
			return fmt.Errorf("%s: get payment by provider payment id: %w", op, err)
		}

		if payment == nil {
			return fmt.Errorf("%s: payment is nil", op)
		}

		if payment.Status == StatusSucceeded {
			return fmt.Errorf("payment already succeeded: %w", ErrPaymentStatusTransitionInvalid)
		}

		err = validateProviderPayment(providerPayment, payment, StatusCanceled)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		orderDetails, err := s.ordersProvider.GetDetailsByID(txCtx, payment.OrderID)
		if err != nil {
			return fmt.Errorf("get order details: %w", err)
		}

		if payment.Status != StatusCanceled {
			if err := s.repo.MarkCanceled(txCtx, payment.ID); err != nil {
				return fmt.Errorf("mark payment canceled: %w", err)
			}
		}

		if orderDetails.Status == orders.StatusCanceled {
			return nil
		}

		if orderDetails.Status != orders.StatusPending {
			return fmt.Errorf("order is not pending: %w", ErrPaymentStatusTransitionInvalid)
		}

		if err := s.ordersProvider.MarkCanceled(txCtx, orderDetails.ID); err != nil {
			return fmt.Errorf("mark order canceled: %w", err)
		}

		reservedItems := reservedItemsFromOrder(orderDetails)

		if err := s.inventoryProvider.ReleaseReservedQuantities(txCtx, reservedItems); err != nil {
			return fmt.Errorf("release reserved inventory: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
