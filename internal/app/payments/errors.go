package payments

import "errors"

var (
	ErrPaymentNotFound                = errors.New("payment not found")
	ErrActivePaymentAlreadyExists     = errors.New("active payment already exists")
	ErrSucceededPaymentAlreadyExists  = errors.New("succeeded payment already exists")
	ErrPaymentStateConflict           = errors.New("payment state conflict")
	ErrProviderRejected               = errors.New("payment provider operation rejected")
	ErrProviderFailure                = errors.New("payment provider operation failed")
	ErrUserIDIsNil                    = errors.New("user id is nil")
	ErrPaymentIDIsNil                 = errors.New("payment id is nil")
	ErrOrderIDIsNil                   = errors.New("order id is nil")
	ErrOrderNotPayable                = errors.New("order not payable")
	ErrOrderExpired                   = errors.New("order expired")
	ErrProviderPaymentIDRequired      = errors.New("provider payment id required")
	ErrPaymentStatusTransitionInvalid = errors.New("payment status transition invalid")
)
