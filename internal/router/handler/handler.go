package handler

import (
	"encoding/json"
	"net/http"
)

type Handler struct{}

type Response struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func NewHandler() *Handler {
	return &Handler{}
}

const (
	StatusOK    = "ok"
	StatusError = "error"
)

func (h *Handler) HealthLive() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, OK())
	}
}

func WriteJSON(w http.ResponseWriter, statusCode int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(v)
}

func OK() Response {
	return Response{
		Status: StatusOK,
	}
}
