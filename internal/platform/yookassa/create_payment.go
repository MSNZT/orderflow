package yookassa

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

func (c *Client) CreatePayment(ctx context.Context, params CreatePaymentParams) (*Payment, error) {
	const op = "yookassa.client.CreatePayment"

	if err := params.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	paymentReq := c.buildCreatePaymentRequest(params)

	path := "payments"
	payment, err := c.doPaymentRequest(ctx, http.MethodPost, path, params.IdempotencyKey.String(), paymentReq, op)
	if err != nil {
		return nil, err
	}

	if err := validateCreatePaymentResponse(payment, paymentReq); err != nil {
		return nil, err
	}

	return payment, nil
}

func (c Client) buildCreatePaymentRequest(params CreatePaymentParams) *createPaymentRequest {
	paymentReq := createPaymentRequest{
		Money: Money{
			Value:    formatAmount(params.AmountCents),
			Currency: strings.TrimSpace(params.Currency),
		},
		Capture:     false,
		Description: params.Description,
		Metadata: Metadata{
			OrderID:   params.OrderID.String(),
			PaymentID: params.LocalPaymentID.String(),
		},
		Confirmation: confirmationRequest{
			Type:      "redirect",
			ReturnURL: c.returnURL,
		},
	}

	return &paymentReq
}

func (p *CreatePaymentParams) validate() error {
	const op = "yookassa.CreatePaymentParams.validate"

	if p.AmountCents <= 0 {
		return fmt.Errorf("%s: %w", op, ErrInvalidArgument)
	}

	if p.OrderID == uuid.Nil || p.LocalPaymentID == uuid.Nil || p.IdempotencyKey == uuid.Nil {
		return fmt.Errorf("%s: %w", op, ErrInvalidArgument)
	}

	if strings.TrimSpace(p.Currency) == "" {
		return fmt.Errorf("%s: %w", op, ErrInvalidArgument)
	}

	return nil
}

func validateCreatePaymentResponse(p *Payment, req *createPaymentRequest) error {
	if p.Money.Value != req.Money.Value {
		return fmt.Errorf(
			"amount mismatch: expected %s, got %s: %w",
			req.Money.Value,
			p.Money.Value,
			ErrInvalidResponse,
		)
	}

	if p.Money.Currency != req.Money.Currency {
		return fmt.Errorf(
			"currency mismatch: expected %s, got %s: %w",
			req.Money.Currency,
			p.Money.Currency,
			ErrInvalidResponse,
		)
	}

	if p.Metadata.OrderID != req.Metadata.OrderID {
		return fmt.Errorf(
			"metadata order_id mismatch: %w",
			ErrInvalidResponse,
		)
	}

	if p.Metadata.PaymentID != req.Metadata.PaymentID {
		return fmt.Errorf(
			"metadata payment_id mismatch: %w",
			ErrInvalidResponse,
		)
	}

	if p.Status == StatusPending {
		if p.Confirmation == nil {
			return fmt.Errorf(
				"pending payment has no confirmation: %w",
				ErrInvalidResponse,
			)
		}

		if p.Confirmation.Type != req.Confirmation.Type {
			return fmt.Errorf(
				"confirmation type mismatch: expected %s, got %s: %w",
				req.Confirmation.Type,
				p.Confirmation.Type,
				ErrInvalidResponse,
			)
		}

		if strings.TrimSpace(p.Confirmation.ConfirmationURL) == "" {
			return fmt.Errorf(
				"pending payment has empty confirmation URL: %w",
				ErrInvalidResponse,
			)
		}
	}

	return nil
}
