package yookassa

import (
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	StatusPending           PaymentStatus = "pending"
	StatusWaitingForCapture PaymentStatus = "waiting_for_capture"
	StatusSucceeded         PaymentStatus = "succeeded"
	StatusCanceled          PaymentStatus = "canceled"
)

type apiErrorResponse struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Parameter   string `json:"parameter"`
}

type CreatePaymentParams struct {
	AmountCents    int64
	Currency       string
	Description    string
	OrderID        uuid.UUID
	LocalPaymentID uuid.UUID
	IdempotencyKey uuid.UUID
}

type Metadata struct {
	OrderID   string `json:"order_id"`
	PaymentID string `json:"payment_id"`
}

type Money struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type Сonfirmation struct {
	Type            string `json:"type"`
	ConfirmationURL string `json:"confirmation_url"`
}

type Recipient struct {
	AccountID string `json:"account_id"`
	GatewayID string `json:"gateway_id"`
}

type CancellationDetails struct {
	Party  string `json:"party"`
	Reason string `json:"reason"`
}

type confirmationRequest struct {
	Type      string `json:"type"`
	ReturnURL string `json:"return_url"`
}

type createPaymentRequest struct {
	Money        Money               `json:"amount"`
	Capture      bool                `json:"capture"`
	Confirmation confirmationRequest `json:"confirmation"`
	Description  string              `json:"description"`
	Metadata     Metadata            `json:"metadata"`
}

type Payment struct {
	ID                  string               `json:"id"`
	Status              PaymentStatus        `json:"status"`
	Paid                bool                 `json:"paid"`
	Money               Money                `json:"amount"`
	Confirmation        *Сonfirmation        `json:"confirmation"`
	CreatedAt           time.Time            `json:"created_at"`
	Description         string               `json:"description"`
	Metadata            Metadata             `json:"metadata"`
	Recipient           Recipient            `json:"recipient"`
	Refundable          bool                 `json:"refundable"`
	Test                bool                 `json:"test"`
	CancellationDetails *CancellationDetails `json:"cancellation_details"`
}
