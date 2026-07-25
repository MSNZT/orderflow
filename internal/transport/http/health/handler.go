package health

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/MSNZT/orderflow/internal/transport/http/httpmw"
	"github.com/MSNZT/orderflow/internal/transport/http/response"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	resp   *response.Response
	log    *slog.Logger
	dbPool *pgxpool.Pool
}

func NewHandler(log *slog.Logger, resp *response.Response, dbPool *pgxpool.Pool) *Handler {
	return &Handler{log: log, resp: resp, dbPool: dbPool}
}

func (h *Handler) Live(w http.ResponseWriter, r *http.Request) error {
	h.resp.JSON(w, http.StatusOK, response.StatusResponse{
		Status: response.StatusOK,
	})

	return nil
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) error {
	const op = "health.handler.Ready"
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()

	if err := h.dbPool.Ping(ctx); err != nil {
		return httpmw.WrapHTTPError(
			http.StatusServiceUnavailable, op,
			"service unavailable",
			fmt.Errorf("ping postgres: %w", err),
		)
	}

	h.resp.JSON(w, http.StatusOK, response.StatusResponse{
		Status: response.StatusOK,
	})
	return nil
}
