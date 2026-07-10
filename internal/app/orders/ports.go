package orders

import (
	"context"
	"time"

	"github.com/MSNZT/orderflow/internal/app/cart"
	"github.com/MSNZT/orderflow/internal/app/inventory"
	"github.com/google/uuid"
)

type Repository interface {
	ListByUserID(ctx context.Context, userID uuid.UUID, offset int, limit int) ([]Order, error)
	GetDetailsByIDAndUserID(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (details *OrderDetails, err error)
	CreateOrder(ctx context.Context, o *Order) error
	CreateOrderItems(ctx context.Context, orderItems []OrderItem) error
	MarkPaid(ctx context.Context, orderID uuid.UUID) error
	MarkCanceled(ctx context.Context, orderID uuid.UUID) error
	MarkExpired(ctx context.Context, orderID uuid.UUID) error
	FindExpiredPendingIDs(ctx context.Context, now time.Time, limit int) ([]uuid.UUID, error)
	GetDetailsByID(ctx context.Context, orderID uuid.UUID) (details *OrderDetails, err error)
}

type CartProvider interface {
	GetSelectedItemsForCheckout(ctx context.Context, userID uuid.UUID, productIDs []uuid.UUID) ([]cart.CheckoutItem, error)
	DeleteSelectedItems(ctx context.Context, userID uuid.UUID, productIDs []uuid.UUID) error
}

type InventoryRepository interface {
	GetByProductIDsForUpdate(ctx context.Context, productIDs []uuid.UUID) ([]inventory.Inventory, error)
	ReserveQuantity(ctx context.Context, productID uuid.UUID, quantity int) error
	ReleaseReservedQuantities(ctx context.Context, reservedItems []inventory.ReservedItem) error
}

type PaymentRepository interface {
	CancelActiveByOrderID(ctx context.Context, orderID uuid.UUID, now time.Time) error
}
