package orders

import "errors"

var (
	ErrOrderNotFound = errors.New("order not found")
	ErrUserIDIsNil   = errors.New("user id is nil")
)
