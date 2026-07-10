package yookassa

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

func (c *Client) GetPaymentByID(ctx context.Context, paymentID string) (*Payment, error) {
	const op = "yookassa.client.GetPaymentByID"

	paymentID = strings.TrimSpace(paymentID)
	if paymentID == "" {
		return nil, fmt.Errorf("%s: invalid payment id: %q: %w", op, paymentID, ErrInvalidArgument)
	}

	path := fmt.Sprintf("%s/%s", "payments", paymentID)
	payment, err := c.doPaymentRequest(ctx, http.MethodPost, path, "", nil, op)
	if err != nil {
		return nil, err
	}

	if err := validateGetPaymentResponse(payment, paymentID); err != nil {
		return nil, err
	}

	return payment, nil
}

func validateGetPaymentResponse(p *Payment, paymentID string) error {
	if err := validatePayment(p); err != nil {
		return err
	}

	if p.ID != paymentID {
		return fmt.Errorf("mismatch payment id: expected %s, got: %s: %w", paymentID, p.ID, ErrInvalidResponse)
	}

	return nil
}
