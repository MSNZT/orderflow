package inventory

import "errors"

var (
	ErrInventoryNotFound        = errors.New("inventory not found")
	ErrInventoryAlreadyExists   = errors.New("inventory already exists")
	ErrInsufficientStock        = errors.New("insufficient stock")
	ErrInventoryQuantityInvalid = errors.New("inventory quantity invalid")
	ErrProductIDIsNil           = errors.New("product id is nil")
	ErrDuplicateProductID       = errors.New("duplicate product id")
	ErrReservedItemsEmpty       = errors.New("reserved items empty")
)
