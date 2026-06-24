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
