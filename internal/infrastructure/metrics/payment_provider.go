package metrics

import (
	"context"
	"time"

	paymentsapp "github.com/MSNZT/orderflow/internal/app/payments"
)

type PaymentOperationsRecorder interface {
	CreateFinished(duration time.Duration, err error)
	CaptureFinished(duration time.Duration, err error)
	CancelFinished(duration time.Duration, err error)
}

type PaymentProviderDecorator struct {
	next     paymentsapp.PaymentProvider
	recorder PaymentOperationsRecorder
}

func NewPaymentProviderDecorator(
	next paymentsapp.PaymentProvider,
	recorder PaymentOperationsRecorder,
) *PaymentProviderDecorator {
	return &PaymentProviderDecorator{
		next:     next,
		recorder: recorder,
	}
}

func (d *PaymentProviderDecorator) CreatePayment(
	ctx context.Context,
	params paymentsapp.ProviderCreateParams,
) (*paymentsapp.ProviderCreateResult, error) {
	startedAt := time.Now()

	result, err := d.next.CreatePayment(ctx, params)

	d.recorder.CreateFinished(
		time.Since(startedAt),
		err,
	)

	return result, err
}

func (d *PaymentProviderDecorator) GetPayment(
	ctx context.Context,
	providerPaymentID string,
) (*paymentsapp.ProviderPayment, error) {
	return d.next.GetPayment(ctx, providerPaymentID)
}

func (d *PaymentProviderDecorator) CapturePayment(
	ctx context.Context,
	input paymentsapp.CapturePaymentInput,
) (*paymentsapp.ProviderPayment, error) {
	startedAt := time.Now()

	result, err := d.next.CapturePayment(ctx, input)

	d.recorder.CaptureFinished(
		time.Since(startedAt),
		err,
	)

	return result, err
}

func (d *PaymentProviderDecorator) CancelPayment(
	ctx context.Context,
	input paymentsapp.CancelPaymentInput,
) (*paymentsapp.ProviderPayment, error) {
	startedAt := time.Now()

	result, err := d.next.CancelPayment(ctx, input)

	d.recorder.CancelFinished(
		time.Since(startedAt),
		err,
	)

	return result, err
}

var _ paymentsapp.PaymentProvider = (*PaymentProviderDecorator)(nil)
