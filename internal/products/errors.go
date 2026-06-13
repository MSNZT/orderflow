package products

import "errors"

var (
	ErrProductNotFound          = errors.New("product not found")
	ErrProductAlreadyExists     = errors.New("product already exists")
	ErrProductNameInvalid       = errors.New("invalid product name")
	ErrProductPriceCentsInvalid = errors.New("product price must be greater than zero")
	ErrProductCurrencyInvalid   = errors.New("invalid product currency")
	ErrInitialQuantityInvalid   = errors.New("invalid initial quantity")
)
