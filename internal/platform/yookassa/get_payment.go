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
	payment, err := c.doPaymentRequest(ctx, http.MethodGet, path, "", nil, op)
	if err != nil {
		return nil, err
	}

	if payment.ID != paymentID {
		return nil, fmt.Errorf(
			"mismatch payment id: expected %s, got: %s: %w", paymentID, payment.ID, ErrInvalidResponse)
	}

	return payment, nil
}
