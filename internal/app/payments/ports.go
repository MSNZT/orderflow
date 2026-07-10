package payments

import (
	"context"
	"time"

	"github.com/MSNZT/orderflow/internal/app/inventory"
	"github.com/MSNZT/orderflow/internal/app/orders"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, payment Payment) (*Payment, error)
	ApplyProviderCreateResult(
		ctx context.Context, paymentID uuid.UUID, result *ProviderCreateResult) (*Payment, error)
	MarkFailed(ctx context.Context, paymentID uuid.UUID) error
	MarkSucceeded(ctx context.Context, paymentID uuid.UUID) error
	MarkCanceled(ctx context.Context, paymentID uuid.UUID) error
	MarkWaitingForCapture(ctx context.Context, paymentID uuid.UUID) error
	GetActiveByOrderID(ctx context.Context, orderID uuid.UUID) (*Payment, error)
	GetByProviderPaymentID(ctx context.Context, providerPaymentID string) (*Payment, error)
	CancelActiveByOrderID(ctx context.Context, orderID uuid.UUID, now time.Time) error
	HasWaitingForCaptureByOrderID(ctx context.Context, orderID uuid.UUID) (bool, error)
}

type OrdersProvider interface {
	GetDetailsByIDAndUserID(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (details *orders.OrderDetails, err error)
	GetDetailsByID(ctx context.Context, orderID uuid.UUID) (details *orders.OrderDetails, err error)
	MarkPaid(ctx context.Context, orderID uuid.UUID) error
	MarkCanceled(ctx context.Context, orderID uuid.UUID) error
	MarkExpired(ctx context.Context, orderID uuid.UUID) error
}

type PaymentProvider interface {
	CreatePayment(ctx context.Context, params ProviderCreateParams) (*ProviderCreateResult, error)
	GetPayment(ctx context.Context, providerPaymentID string) (*ProviderPayment, error)
	CapturePayment(ctx context.Context, input CapturePaymentInput) (*ProviderPayment, error)
	CancelPayment(ctx context.Context, input CancelPaymentInput) (*ProviderPayment, error)
}

type InventoryProvider interface {
	CommitReservedQuantities(ctx context.Context, reservedItems []inventory.ReservedItem) error
	ReleaseReservedQuantities(ctx context.Context, reservedItems []inventory.ReservedItem) error
}
