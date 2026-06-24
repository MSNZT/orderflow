package payments

import "errors"

var (
	ErrPaymentNotFound               = errors.New("payment not found")
	ErrActivePaymentAlreadyExists    = errors.New("active payment already exists")
	ErrSucceededPaymentAlreadyExists = errors.New("succeeded payment already exists")
)
