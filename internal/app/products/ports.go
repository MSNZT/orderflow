package products

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	ListActive(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Product, error)
	Create(ctx context.Context, p *Product) (*Product, error)
}

type InventoryRepository interface {
	Create(ctx context.Context, productID uuid.UUID, quantity int32) error
}

type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
