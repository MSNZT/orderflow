package orders

import (
	"time"

	"github.com/google/uuid"
)

type Status string
type Currency string

const (
	StatusPending   Status = "pending"
	StatusPaid      Status = "paid"
	StatusCancelled Status = "cancelled"
)

type Order struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Status          Status
	TotalPriceCents int64
	Currency        string
	UpdatedAt       time.Time
	CreatedAt       time.Time
}

type OrderItem struct {
	ID                  uuid.UUID
	OrderID             uuid.UUID
	ProductID           uuid.UUID
	ProductName         string
	UnitPriceCents      int64
	Currency            string
	Quantity            int
	LineTotalPriceCents int64
	CreatedAt           time.Time
}
