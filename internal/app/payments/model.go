package payments

import (
	"time"

	"github.com/google/uuid"
)

type Status string

type PaymentAction string

const (
	paymentActionCapture PaymentAction = "capture"
	paymentActionCancel  PaymentAction = "cancel"
	paymentActionExpired PaymentAction = "expired"
)

const (
	StatusCreating          Status = "creating"
	StatusPending           Status = "pending"
	StatusWaitingForCapture Status = "waiting_for_capture"
	StatusSucceeded         Status = "succeeded"
	StatusCanceled          Status = "canceled"
	StatusFailed            Status = "failed"
)

type ProviderCreateResult struct {
	ProviderPaymentID string
	Status            Status
	ConfirmationURL   *string
	Test              bool
	ProviderCreatedAt time.Time
}

type ProviderPayment struct {
	ID          string
	Status      Status
	AmountCents int64
	Currency    string
}

type ProviderCreateParams struct {
	AmountCents    int64
	Currency       string
	Description    string
	OrderID        uuid.UUID
	LocalPaymentID uuid.UUID
	IdempotencyKey uuid.UUID
}

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

type Payment struct {
	ID                 uuid.UUID
	OrderID            uuid.UUID
	ProviderPaymentID  *string
	IdempotencyKey     uuid.UUID
	Status             Status
	AmountCents        int64
	Currency           string
	ConfirmationURL    *string
	Test               *bool
	CancellationParty  *string
	CancellationReason *string
	ProviderCreatedAt  *time.Time
	SucceededAt        *time.Time
	CanceledAt         *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
