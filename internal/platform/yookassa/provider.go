package yookassa

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/MSNZT/orderflow/internal/app/payments"
)

type Provider struct {
	client *Client
}

var _ payments.PaymentProvider = (*Provider)(nil)

func NewProvider(client *Client) *Provider {
	return &Provider{client: client}
}

func (p *Provider) CreatePayment(
	ctx context.Context, params payments.ProviderCreateParams) (*payments.ProviderCreateResult, error) {
	yookassaParams := CreatePaymentParams{
		AmountCents:    params.AmountCents,
		Currency:       params.Currency,
		Description:    params.Description,
		OrderID:        params.OrderID,
		LocalPaymentID: params.LocalPaymentID,
		IdempotencyKey: params.IdempotencyKey,
	}

	payment, err := p.client.CreatePayment(ctx, yookassaParams)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, errors.Join(payments.ErrProviderFailure, ctxErr)
		}

		switch {
		case errors.Is(err, ErrInvalidArgument),
			errors.Is(err, ErrInvalidRequest),
			errors.Is(err, ErrInvalidCredentials),
			errors.Is(err, ErrForbidden):
			return nil, errors.Join(payments.ErrProviderRejected, err)
		default:
			return nil, errors.Join(payments.ErrProviderFailure, err)
		}
	}

	if payment == nil {
		return nil, fmt.Errorf(
			"yookassa provider: client returned nil payment without error: %w",
			payments.ErrProviderFailure,
		)
	}

	status, err := mapPaymentStatus(payment.Status)
	if err != nil {
		return nil, err
	}

	var confirmationURL *string
	if payment.Confirmation != nil {
		trimmedURL := strings.TrimSpace(payment.Confirmation.ConfirmationURL)

		if trimmedURL != "" {
			confirmationURL = &trimmedURL
		}
	}

	res := payments.ProviderCreateResult{
		ProviderPaymentID: payment.ID,
		Status:            status,
		ConfirmationURL:   confirmationURL,
		Test:              payment.Test,
		ProviderCreatedAt: payment.CreatedAt,
	}
	return &res, nil
}

func mapPaymentStatus(status PaymentStatus) (payments.Status, error) {
	switch status {
	case StatusPending:
		return payments.StatusPending, nil
	case StatusWaitingForCapture:
		return payments.StatusWaitingForCapture, nil
	case StatusSucceeded:
		return payments.StatusSucceeded, nil
	case StatusCanceled:
		return payments.StatusCanceled, nil
	}

	return "", fmt.Errorf("unsupported yookassa payment status %q: %w", status, payments.ErrProviderFailure)
}
