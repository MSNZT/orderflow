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
	ProductID  uuid.UUID
	Name       string
	Quantity   int32
	PriceCents int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
