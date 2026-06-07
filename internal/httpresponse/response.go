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

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
