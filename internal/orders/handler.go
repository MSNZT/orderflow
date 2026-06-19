package orders

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/MSNZT/orderflow/internal/authcontext"
	"github.com/MSNZT/orderflow/internal/httpresponse"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
	log     *slog.Logger
}

type createOrderRequest struct {
	ProductIDs []uuid.UUID `json:"productIDs"`
}

type createOrderResponse struct {
	ID              uuid.UUID `json:"id"`
	Status          Status    `json:"status"`
	TotalPriceCents int64     `json:"total_price_cents"`
	Currency        string    `json:"currency"`
	CreatedAt       time.Time `json:"created_at"`
}

func NewHandler(log *slog.Logger, service *Service) *Handler {
	return &Handler{service: service, log: log}
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	const op = "orders.handler.CreateOrder"

	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		httpresponse.Unauthorized(w)
		return
	}

	var reqCreateOrder createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&reqCreateOrder); err != nil {
		httpresponse.BadRequest(w)
		return
	}

	order, err := h.service.CreateOrder(r.Context(), userID, reqCreateOrder.ProductIDs)
	if err != nil {
		fmt.Println("Ошибка", err)
		return
	}

	fmt.Println("------===---", order)

	res := createOrderResponse{
		ID:              order.ID,
		Status:          order.Status,
		TotalPriceCents: order.TotalPriceCents,
		Currency:        order.Currency,
		CreatedAt:       order.CreatedAt,
	}

	if err := httpresponse.JSON(w, http.StatusCreated, res); err != nil {
		h.log.Error("failed to send create order response", slog.String("op", op), slog.String("err", err.Error()))
		httpresponse.InternalError(w)
		return
	}
}
