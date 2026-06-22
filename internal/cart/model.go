package cart

import (
	"time"

	"github.com/google/uuid"
)

type Cart struct {
	Items           []CartItem
	TotalPriceCents int64
}

type CartItem struct {
	ProductID           uuid.UUID
	Name                string
	Quantity            int32
	PriceCents          int64
	LineTotalPriceCents int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type CheckoutItem struct {
	ProductID           uuid.UUID
	ProductName         string
	UnitPriceCents      int64
	Currency            string
	Quantity            int
	LineTotalPriceCents int64
	IsProductActive     bool
}
