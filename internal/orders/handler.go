package orders

import (
	"encoding/json"
	"errors"
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
	ProductIDs []uuid.UUID `json:"product_ids"`
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
		mapErrors(w, err, h.log, op)
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

func mapErrors(w http.ResponseWriter, err error, log *slog.Logger, op string) {
	switch {
	case errors.Is(err, ErrUserIDIsNil):
		httpresponse.Unauthorized(w)
		return
	case errors.Is(err, ErrProductIDsEmpty):
		httpresponse.BadRequestMsg(w, "product_ids must not be empty")
		return
	case errors.Is(err, ErrProductIDIsNil):
		httpresponse.BadRequestMsg(w, "product_ids contains an empty UUID")
	case errors.Is(err, ErrDuplicateProductID):
		httpresponse.BadRequestMsg(w, "product_ids contains duplicates")
		return
	case errors.Is(err, ErrCartChanged):
		httpresponse.Error(
			w,
			http.StatusConflict,
			"cart contents changed, refresh the cart and try again",
		)
		return
	case errors.Is(err, ErrProductInactive):
		httpresponse.Error(
			w,
			http.StatusConflict,
			"one or more selected products are unavailable",
		)
		return
	case errors.Is(err, ErrCurrencyMismatch):
		httpresponse.Error(
			w,
			http.StatusConflict,
			"selected products have different currencies",
		)
		return
	case errors.Is(err, ErrInventoryNotFound):
		log.Error(
			"inventory not found while creating order",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		httpresponse.Error(
			w,
			http.StatusConflict,
			"one or more selected products are unavailable",
		)
	case errors.Is(err, ErrInsufficientStock):
		httpresponse.Error(
			w,
			http.StatusConflict,
			"insufficient stock for one or more selected products",
		)
		return
	default:
		log.Error("failed to create order", slog.String("op", op), slog.String("err", err.Error()))
		httpresponse.InternalError(w)
		return
	}
}
