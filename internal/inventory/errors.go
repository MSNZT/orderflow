package inventory

import "errors"

var (
	ErrInventoryNotFound      = errors.New("inventory not found")
	ErrInventoryAlreadyExists = errors.New("inventory already exists")
)
