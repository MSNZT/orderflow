package payments

import (
	"time"

	"github.com/google/uuid"
)

type Status string

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

type ProviderCreateParams struct {
	AmountCents    int64
	Currency       string
	Description    string
	OrderID        uuid.UUID
	LocalPaymentID uuid.UUID
	IdempotencyKey uuid.UUID
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
