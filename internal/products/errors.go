package products

import "errors"

var (
	ErrProductNotFound          = errors.New("product not found")
	ErrProductAlreadyExists     = errors.New("product already exists")
	ErrProductPriceCentsInvalid = errors.New("product price must be greater than zero")
	ErrProductCurrencyInvalid   = errors.New("invalid product currency")
)
