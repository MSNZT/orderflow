package payments

import (
	"context"

	"github.com/MSNZT/orderflow/internal/app/orders"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, payment Payment) (*Payment, error)
	ApplyProviderCreateResult(
		ctx context.Context, paymentID uuid.UUID, result *ProviderCreateResult) (*Payment, error)
	MarkFailed(ctx context.Context, paymentID uuid.UUID) error
}

type OrdersProvider interface {
	GetDetailsByIDAndUserID(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (details *orders.OrderDetails, err error)
}

type PaymentProvider interface {
	CreatePayment(ctx context.Context, params ProviderCreateParams) (*ProviderCreateResult, error)
}
