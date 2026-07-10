package payments

import (
	"context"
	"fmt"
	"strings"

	"github.com/MSNZT/orderflow/internal/app/orders"
)

func (s *Service) ProcessSucceededPayment(ctx context.Context, providerPayment ProviderPayment) error {
	const op = "payments.service.ProcessSucceededPayment"

	providerPaymentID := strings.TrimSpace(providerPayment.ID)
	if providerPaymentID == "" {
		return fmt.Errorf("%s: %w", op, ErrProviderPaymentIDRequired)
	}

	err := s.txManager.WithinTx(ctx, func(txCtx context.Context) error {
		payment, err := s.repo.GetByProviderPaymentID(txCtx, providerPaymentID)
		if err != nil {
			return fmt.Errorf("get payment by provider payment id: %w", err)
		}

		if payment == nil {
			return fmt.Errorf("%s: payment is nil", op)
		}

		if payment.Status == StatusCanceled {
			return fmt.Errorf("payment already canceled: %w", ErrPaymentStatusTransitionInvalid)
		}

		err = validateProviderPayment(providerPayment, payment, StatusSucceeded)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		orderDetails, err := s.ordersProvider.GetDetailsByID(txCtx, payment.OrderID)
		if err != nil {
			return fmt.Errorf("get order details: %w", err)
		}

		if orderDetails.Order.Status == orders.StatusPaid {
			return nil
		}

		if orderDetails.Status != orders.StatusPending {
			return fmt.Errorf("order is not pending: %w", ErrPaymentStatusTransitionInvalid)
		}

		if payment.Status != StatusSucceeded {
			if err := s.repo.MarkSucceeded(txCtx, payment.ID); err != nil {
				return fmt.Errorf("mark payment succeeded: %w", err)
			}
		}

		if err := s.ordersProvider.MarkPaid(txCtx, orderDetails.Order.ID); err != nil {
			return fmt.Errorf("mark order paid: %w", err)
		}

		reservedItems := reservedItemsFromOrder(orderDetails)

		err = s.inventoryProvider.CommitReservedQuantities(txCtx, reservedItems)
		if err != nil {
			return fmt.Errorf("commit reserved inventory: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
