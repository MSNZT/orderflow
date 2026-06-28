package cart

import (
	"context"

	"github.com/MSNZT/orderflow/internal/app/products"
	"github.com/google/uuid"
)

type Repository interface {
	GetItems(ctx context.Context, userId uuid.UUID, limit int, offset int) ([]CartItem, error)
	GetOrCreateByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
	AddItem(ctx context.Context, cartID uuid.UUID, productID uuid.UUID, quantity int32) error
	UpdateItemQuantity(
		ctx context.Context, cartID uuid.UUID, productID uuid.UUID, quantity int32) error
	DeleteItem(ctx context.Context, cartID uuid.UUID, productID uuid.UUID) error
	ClearItems(ctx context.Context, cartID uuid.UUID) error
	GetSelectedItemsForCheckout(ctx context.Context, cartID uuid.UUID, productIDs []uuid.UUID) ([]CheckoutItem, error)
	DeleteSelectedItems(ctx context.Context, cartID uuid.UUID, productIDs []uuid.UUID) (int64, error)
}

type ProductsProvider interface {
	GetByID(ctx context.Context, productID uuid.UUID) (*products.Product, error)
}
