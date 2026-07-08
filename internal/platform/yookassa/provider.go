package yookassa

import (
	"context"
	"errors"
	"fmt"
	"strconv"
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

func (p *Provider) GetPayment(ctx context.Context, providerPaymentID string) (*payments.ProviderPayment, error) {
	const op = "yookassa.provider.GetPayment"

	payment, err := p.client.GetPaymentByID(ctx, providerPaymentID)
	if err != nil {
		return nil, fmt.Errorf("%s: get payment by provider payment id: %w", op, err)
	}

	status, err := mapPaymentStatus(payment.Status)
	if err != nil {
		return nil, fmt.Errorf("%s: map payment status: %w", op, err)
	}

	amountCents, err := parseAmountCents(payment.Money.Value)
	if err != nil {
		return nil, fmt.Errorf("%s: parse amount: %w", op, err)
	}

	return &payments.ProviderPayment{
		ID:          payment.ID,
		Status:      status,
		AmountCents: amountCents,
		Currency:    payment.Money.Currency,
	}, nil
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

func parseAmountCents(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if len(value) == 0 {
		return 0, fmt.Errorf("invalid value: empty amount")
	}

	if strings.HasPrefix(value, "-") {
		return 0, fmt.Errorf("invalid value: negative number not allowed")
	}

	parts := strings.Split(value, ".")

	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid value: too many dots")
	}

	intPart := parts[0]
	if intPart == "" {
		return 0, fmt.Errorf("invalid value: missing integer part")
	}

	fracPart := ""

	if len(parts) == 2 {
		fracPart = parts[1]

		if len(fracPart) == 0 {
			return 0, fmt.Errorf("invalid value: missing fractional part")
		}

		if len(fracPart) >= 3 {
			return 0, fmt.Errorf("invalid value: too many decimal places, maximum 2 allowed")
		}

		if len(fracPart) == 1 {
			fracPart += "0"
		}
	} else {
		fracPart = "00"
	}

	for _, ch := range intPart + fracPart {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid value: contains non-digit characters")
		}
	}

	amount, err := strconv.ParseInt(intPart+fracPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid value: failed to parse amount: %w", err)
	}

	return amount, nil
}
