package router

import (
	"encoding/json"
	"net/http"
)

type Handler struct{}

type Response struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

func NewHandler() *Handler {
	return &Handler{}
}

const (
	StatusOK    = "ok"
	StatusError = "error"
)

func (h *Handler) HealthLive(w http.ResponseWriter, r *http.Request) {
	_ = WriteJSON(w, http.StatusOK, HealthResponse{
		Status: StatusOK,
	})
}

func WriteJSON(w http.ResponseWriter, statusCode int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(v)
}
