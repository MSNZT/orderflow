package orders

import "errors"

var (
	ErrOrdersNotFound = errors.New("orders not found")
	ErrUserIDIsNil    = errors.New("user id is nil")
)
