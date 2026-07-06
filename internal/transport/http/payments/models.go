package payments

import (
	"github.com/MSNZT/orderflow/internal/app/payments"
	"github.com/google/uuid"
)

type createPaymentResponse struct {
	ID              uuid.UUID       `json:"id"`
	OrderID         uuid.UUID       `json:"order_id"`
	Status          payments.Status `json:"status"`
	AmountCents     int64           `json:"amount_cents"`
	Currency        string          `json:"currency"`
	ConfirmationURL *string         `json:"confirmation_url,omitempty"`
}
