package inventory

import (
	"time"

	"github.com/google/uuid"
)

type Inventory struct {
	ID               uuid.UUID
	Quantity         int32
	ReservedQuantity int32
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
