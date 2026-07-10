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
		return nil, mapProviderError(ctx, err)
	}

	status, err := toDomainPaymentStatus(payment.Status)
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
		return nil, mapProviderError(ctx, err)
	}

	providerPayment, err := mapProviderPayment(payment)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return providerPayment, nil
}

func (p *Provider) CapturePayment(ctx context.Context, input payments.CapturePaymentInput) (*payments.ProviderPayment, error) {
	const op = "yookassa.provider.CapturePayment"
	req := CapturePaymentInput{
		ProviderPaymentID: input.ProviderPaymentID,
		IdempotencyKey:    input.IdempotencyKey,
		AmountCents:       input.AmountCents,
		Currency:          input.Currency,
	}

	payment, err := p.client.CapturePayment(ctx, req)
	if err != nil {
		return nil, mapProviderError(ctx, err)
	}

	providerPayment, err := mapProviderPayment(payment)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return providerPayment, nil
}

func (p *Provider) CancelPayment(ctx context.Context, input payments.CancelPaymentInput) (*payments.ProviderPayment, error) {
	const op = "yookassa.provider.CancelPayment"
	req := CancelPaymentInput{
		ProviderPaymentID: input.ProviderPaymentID,
		IdempotencyKey:    input.IdempotencyKey,
	}

	payment, err := p.client.CancelPayment(ctx, req)
	if err != nil {
		return nil, mapProviderError(ctx, err)
	}

	providerPayment, err := mapProviderPayment(payment)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return providerPayment, nil
}

func toDomainPaymentStatus(status PaymentStatus) (payments.Status, error) {
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

func mapProviderPayment(payment *Payment) (*payments.ProviderPayment, error) {
	if payment == nil {
		return nil, fmt.Errorf(
			"yookassa provider: client returned nil payment without error: %w",
			payments.ErrProviderFailure,
		)
	}

	status, err := toDomainPaymentStatus(payment.Status)
	if err != nil {
		return nil, fmt.Errorf("map payment status: %w", err)
	}

	amountCents, err := parseAmountCents(payment.Money.Value)
	if err != nil {
		return nil, fmt.Errorf("parse amount: %w", err)
	}

	providerPayment := payments.ProviderPayment{
		ID:          payment.ID,
		Status:      status,
		AmountCents: amountCents,
		Currency:    payment.Money.Currency,
	}

	return &providerPayment, nil
}

func mapProviderError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return errors.Join(payments.ErrProviderFailure, ctxErr)
	}

	switch {
	case errors.Is(err, ErrInvalidArgument),
		errors.Is(err, ErrInvalidRequest),
		errors.Is(err, ErrInvalidCredentials),
		errors.Is(err, ErrForbidden):
		return errors.Join(payments.ErrProviderRejected, err)
	default:
		return errors.Join(payments.ErrProviderFailure, err)
	}
}
