package httpresponse

import (
	"encoding/json"
	"net/http"
)

const (
	StatusOK    = "ok"
	StatusError = "error"
)

type StatusResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func JSON(w http.ResponseWriter, statusCode int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(v)
}

func Error(w http.ResponseWriter, statusCode int, message string) error {
	return JSON(w, statusCode, StatusResponse{
		Status:  "error",
		Message: message,
	})
}

func BadRequest(w http.ResponseWriter) {
	Error(w, http.StatusBadRequest, "bad request")
}

func BadRequestMsg(w http.ResponseWriter, message string) {
	if message == "" {
		message = "bad request"
	}
	Error(w, http.StatusBadRequest, message)
}

func Unauthorized(w http.ResponseWriter) {
	Error(w, http.StatusUnauthorized, "unauthorized")
}

func InternalError(w http.ResponseWriter) {
	Error(w, http.StatusInternalServerError, "internal server error")
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
