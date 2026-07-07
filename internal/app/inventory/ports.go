package inventory

import "context"

type ReservedStockRepository interface {
	CommitReservedQuantities(ctx context.Context, reservedItems []ReservedItem) error
	ReleaseReservedQuantities(ctx context.Context, reservedItems []ReservedItem) error
}
