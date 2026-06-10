package products

import (
	"time"

	"github.com/google/uuid"
)

type Product struct {
	Id          uuid.UUID
	Name        string
	Description *string
	PriceCents  int64
	Currency    string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
