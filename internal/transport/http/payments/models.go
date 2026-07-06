package payments

import (
	paymentsapp "github.com/MSNZT/orderflow/internal/app/payments"
	"github.com/google/uuid"
)

type createPaymentResponse struct {
	ID              uuid.UUID          `json:"id"`
	OrderID         uuid.UUID          `json:"order_id"`
	Status          paymentsapp.Status `json:"status"`
	AmountCents     int64              `json:"amount_cents"`
	Currency        string             `json:"currency"`
	ConfirmationURL *string            `json:"confirmation_url,omitempty"`
}
