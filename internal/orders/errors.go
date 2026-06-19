package orders

import "errors"

var (
	ErrOrderNotFound      = errors.New("order not found")
	ErrUserIDIsNil        = errors.New("user id is nil")
	ErrProductIDIsNil     = errors.New("product id is nil")
	ErrProductIDsEmpty    = errors.New("product ids empty")
	ErrDuplicateProductID = errors.New("duplicate product id")
	ErrCartChanged        = errors.New("cart changed")
	ErrProductInactive    = errors.New("product inactive")
	ErrInventoryNotFound  = errors.New("inventory not found")
	ErrInsufficientStock  = errors.New("insufficient stock")
	ErrGenerateUUID       = errors.New("failed to generate uuid")
	ErrCurrencyMismatch   = errors.New("currency mismatch")
)
