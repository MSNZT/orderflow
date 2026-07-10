package yookassa

import (
	"fmt"
	"strings"
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

type PaymentAction string

const (
	ActionCapture PaymentAction = "capture"
	ActionCancel  PaymentAction = "cancel"
)

type CapturePaymentInput struct {
	ProviderPaymentID string
	IdempotencyKey    string
	AmountCents       int64
	Currency          string
}

type CancelPaymentInput struct {
	ProviderPaymentID string
	IdempotencyKey    string
}

type paymentActionRequest struct {
	Amount Money `json:"amount"`
}

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

type Confirmation struct {
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
	Confirmation        *Confirmation        `json:"confirmation"`
	CreatedAt           time.Time            `json:"created_at"`
	Description         string               `json:"description"`
	Metadata            Metadata             `json:"metadata"`
	Recipient           Recipient            `json:"recipient"`
	Refundable          bool                 `json:"refundable"`
	Test                bool                 `json:"test"`
	CancellationDetails *CancellationDetails `json:"cancellation_details"`
}

func validatePayment(p *Payment) error {
	if p == nil {
		return fmt.Errorf("nil payment response: %w", ErrInvalidResponse)
	}

	if strings.TrimSpace(p.ID) == "" {
		return fmt.Errorf("empty payment id: %w", ErrInvalidResponse)
	}

	if !p.Status.Valid(p.Status) {
		return fmt.Errorf("unknown payment status: %q: %w", p.Status, ErrInvalidResponse)
	}

	if strings.TrimSpace(p.Money.Currency) == "" {
		return fmt.Errorf("empty payment currency: %s: %w", p.Money.Currency, ErrInvalidResponse)
	}

	if strings.TrimSpace(p.Money.Value) == "" {
		return fmt.Errorf("empty payment money value: %s: %w", p.Money.Value, ErrInvalidResponse)
	}

	if p.CreatedAt.IsZero() {
		return fmt.Errorf("empty payment created_at: %w", ErrInvalidResponse)
	}

	return nil
}

func (s PaymentStatus) Valid(status PaymentStatus) bool {
	switch status {
	case StatusPending,
		StatusSucceeded,
		StatusCanceled,
		StatusWaitingForCapture:
		return true
	}
	return false
}
