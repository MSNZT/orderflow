package httpmw

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/MSNZT/orderflow/internal/infrastructure/logger"
	"github.com/MSNZT/orderflow/internal/transport/http/response"
)

type HandlerWrapper struct {
	log  *slog.Logger
	resp *response.Response
}

func NewHandlerWrapper(log *slog.Logger, resp *response.Response) *HandlerWrapper {
	return &HandlerWrapper{log: log, resp: resp}
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

func (h *HandlerWrapper) Wrap(next HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := next(w, r)
		if err == nil {
			return
		}

		var responseErr ResponseError
		if errors.As(err, &responseErr) {
			status := responseErr.HTTPStatus()
			if status >= http.StatusInternalServerError {
				h.log.Error("request failed",
					logger.Op(responseErr.Op()),
					slog.Int("status", responseErr.HTTPStatus()),
					logger.Err(err))
			}

			h.resp.Error(w, status, responseErr.PublicMessage())
			return
		}

		h.log.Error("unhandled request error", logger.Err(err))
		h.resp.InternalError(w)
	}
}

type ResponseError interface {
	error
	HTTPStatus() int
	PublicMessage() string
	Op() string
}

type HTTPError struct {
	status        int
	publicMessage string
	op            string
	err           error
}

func (h *HTTPError) Error() string {
	if h.err != nil {
		return h.err.Error()
	}

	return h.publicMessage
}

func (h *HTTPError) HTTPStatus() int {
	return h.status
}

func (h *HTTPError) Unwrap() error {
	return h.err
}

func (h *HTTPError) Op() string {
	return h.op
}

func (h *HTTPError) PublicMessage() string {
	return h.publicMessage
}

func NewHTTPError(status int, op string, publicMessage string) *HTTPError {
	return &HTTPError{status: status, op: op, publicMessage: publicMessage}
}

func WrapHTTPError(
	status int,
	op string,
	publicMessage string,
	err error,
) *HTTPError {
	return &HTTPError{
		status:        status,
		op:            op,
		publicMessage: publicMessage,
		err:           err,
	}
}
