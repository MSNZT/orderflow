package yookassa

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrInvalidArgument      = errors.New("invalid yookassa argument")
	ErrInvalidRequest       = errors.New("invalid yookassa request")
	ErrInvalidResponse      = errors.New("invalid yookassa response")
	ErrInvalidCredentials   = errors.New("invalid yookassa credentials")
	ErrForbidden            = errors.New("operation yookassa forbidden")
	ErrNotFound             = errors.New("resource not found")
	ErrRateLimited          = errors.New("rate limited")
	ErrResultUnknown        = errors.New("operation result unknown")
	ErrUnexpectedResponse   = errors.New("unexpected response")
	ErrProviderUnavailable  = errors.New("yookassa provider unavailable")
	ErrProviderPaymentEmpty = errors.New("provider payment id empty")
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

func mapHTTPStatusError(statusCode int) error {
	switch {
	case statusCode >= http.StatusInternalServerError && statusCode <= 599:
		return ErrResultUnknown

	case statusCode == http.StatusBadRequest:
		return ErrInvalidRequest
	case statusCode == http.StatusUnauthorized:
		return ErrInvalidCredentials
	case statusCode == http.StatusForbidden:
		return ErrForbidden
	case statusCode == http.StatusNotFound:
		return ErrNotFound
	case statusCode == http.StatusTooManyRequests:
		return ErrRateLimited

	default:
		return ErrUnexpectedResponse
	}
}
