package orders

import (
	"time"

	"github.com/google/uuid"
)

const (
	orderStatusPending   = "pending"
	orderStatusPaid      = "paid"
	orderStatusCancelled = "cancelled"
)

type Status string

type Order struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Status          Status
	TotalPriceCents int64
	Currency        string
	UpdatedAt       time.Time
	CreatedAt       time.Time
}

type OrderItems struct {
	ID              uuid.UUID
	OrderID         uuid.UUID
	ProductID       uuid.UUID
	ProductName     string
	UnitPriceCents  int64
	Currency        string
	Quantity        int
	TotalPriceCents int64
	CreatedAt       time.Time
}
