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

func NewHandler() *Handler {
	return &Handler{}
}

const (
	StatusOK    = "ok"
	StatusError = "error"
)

func (h *Handler) HealthLive(w http.ResponseWriter, r *http.Request) {
	err := WriteJSON(w, http.StatusOK, OK())
	if err != nil {
		// позже залогирую, если потребуется
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
