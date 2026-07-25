package response

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/MSNZT/orderflow/internal/infrastructure/logger"
)

const (
	StatusOK    = "ok"
	StatusError = "error"
)

type Response struct {
	log *slog.Logger
}

func New(log *slog.Logger) *Response {
	if log == nil {
		panic("response.New: logger is nil")
	}
	return &Response{log: log}
}

type StatusResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func (r *Response) JSON(w http.ResponseWriter, statusCode int, v any) {
	buf, err := json.Marshal(v)
	if err != nil {
		r.log.Error("failed to marshal json response", logger.Err(err))
		r.InternalError(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if _, err := w.Write(buf); err != nil {
		r.log.Error("failed to write response body to network", logger.Err(err))
	}
}

func (r *Response) Error(w http.ResponseWriter, statusCode int, message string) {
	r.JSON(w, statusCode, StatusResponse{
		Status:  StatusError,
		Message: message,
	})
}

func (r *Response) BadRequest(w http.ResponseWriter) {
	r.Error(w, http.StatusBadRequest, "bad request")
}

func (r *Response) BadRequestMsg(w http.ResponseWriter, message string) {
	if message == "" {
		message = "bad request"
	}
	r.Error(w, http.StatusBadRequest, message)
}

func (r *Response) Unauthorized(w http.ResponseWriter) {
	r.Error(w, http.StatusUnauthorized, "unauthorized")
}

func (r *Response) InternalError(w http.ResponseWriter) {
	r.Error(w, http.StatusInternalServerError, "internal server error")
}

func (r *Response) NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
