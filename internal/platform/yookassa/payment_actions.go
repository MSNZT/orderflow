package yookassa

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

func (c *Client) CapturePayment(ctx context.Context, input CapturePaymentInput) (*Payment, error) {
	const op = "yookassa.client.CapturePayment"

	if input.AmountCents <= 0 {
		return nil, fmt.Errorf("%s: amount cents %w", op, ErrInvalidArgument)
	}
	if strings.TrimSpace(input.Currency) == "" {
		return nil, fmt.Errorf("%s: currency %w", op, ErrInvalidArgument)
	}

	amountCents := formatAmount(input.AmountCents)
	body := paymentActionRequest{
		Amount: Money{
			Value:    amountCents,
			Currency: input.Currency,
		},
	}

	return c.executePaymentAction(ctx, input.ProviderPaymentID, input.IdempotencyKey, body, ActionCapture, op)
}

func (c *Client) CancelPayment(ctx context.Context, input CancelPaymentInput) (*Payment, error) {
	const op = "yookassa.client.CancelPayment"

	return c.executePaymentAction(ctx, input.ProviderPaymentID, input.IdempotencyKey, nil, ActionCancel, op)
}

func (c *Client) executePaymentAction(
	ctx context.Context, providerPaymentID string, idempotencyKey string, body any,
	action PaymentAction, op string) (*Payment, error) {
	providerPaymentID = strings.TrimSpace(providerPaymentID)
	if providerPaymentID == "" {
		return nil, fmt.Errorf("%s: provider payment id: %w", op, ErrInvalidArgument)
	}

	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return nil, fmt.Errorf("%s: idempotency key: %w", op, ErrInvalidArgument)
	}

	path := fmt.Sprintf("%s/%s/%s", "payments", providerPaymentID, action)
	return c.doPaymentRequest(ctx, http.MethodPost, path, idempotencyKey, body, op)
}
