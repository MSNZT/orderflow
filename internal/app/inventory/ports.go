package inventory

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, productID uuid.UUID, quantity int32) error
	GetByProductID(ctx context.Context, productID uuid.UUID) (*Inventory, error)
	GetByProductIDsForUpdate(ctx context.Context, productIDs []uuid.UUID) ([]Inventory, error)
	ReserveQuantity(ctx context.Context, productID uuid.UUID, quantity int) error
	DecreaseQuantity(ctx context.Context, productID uuid.UUID, requestedQuantity int) error
}
