package inventory

import (
	"time"

	"github.com/google/uuid"
)

type Inventory struct {
	ProductID        uuid.UUID
	Quantity         int32
	ReservedQuantity int32
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type StockDecrease struct {
	ProductID uuid.UUID
	Quantity  int32
}

type ReservedItem struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int32     `json:"quantity"`
}
