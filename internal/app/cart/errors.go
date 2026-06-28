package cart

import "errors"

var (
	ErrUserIDIsNil         = errors.New("user id is nil")
	ErrProductIDIsNil      = errors.New("product id is required")
	ErrCartNotFound        = errors.New("cart not found")
	ErrCartItemNotFound    = errors.New("cart item not found")
	ErrQuantityInvalid     = errors.New("quantity must be greater than zero")
	ErrProductNotAvailable = errors.New("product is not available")
)
