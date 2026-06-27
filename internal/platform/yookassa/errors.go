package yookassa

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidArgument    = errors.New("invalid yookassa argument")
	ErrInvalidRequest     = errors.New("invalid yookassa request")
	ErrInvalidResponse    = errors.New("invalid yookassa response")
	ErrInvalidCredentials = errors.New("invalid yookassa credentials")
	ErrForbidden          = errors.New("operation yookassa forbidden")
	ErrNotFound           = errors.New("resource not found")
	ErrRateLimited        = errors.New("rate limited")
	ErrResultUnknown      = errors.New("operation result unknown")
	ErrUnexpectedResponse = errors.New("unexpected response")
)

type APIError struct {
	StatusCode  int
	ID          string
	Code        string
	Description string
	Parameter   string
	Cause       error
}

func (e *APIError) Error() string {
	if e.Parameter != "" {
		return fmt.Sprintf(
			"Yookassa api error: status=%d code=%s parameter=%s description=%s",
			e.StatusCode,
			e.Code,
			e.Parameter,
			e.Description,
		)
	}

	return fmt.Sprintf(
		"Yookassa api error: status=%d code=%s description=%s",
		e.StatusCode,
		e.Code,
		e.Description,
	)
}

func (e *APIError) Unwrap() error {
	return e.Cause
}
