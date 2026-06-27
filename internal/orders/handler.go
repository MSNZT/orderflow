package orders

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/MSNZT/orderflow/internal/transport/http/authcontext"
	"github.com/MSNZT/orderflow/internal/transport/http/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
	log     *slog.Logger
}

type createOrderRequest struct {
	ProductIDs []uuid.UUID `json:"product_ids"`
}

type orderBaseInfo struct {
	ID              uuid.UUID `json:"id"`
	Status          Status    `json:"status"`
	TotalPriceCents int64     `json:"total_price_cents"`
	Currency        string    `json:"currency"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type createOrderResponse struct {
	ID              uuid.UUID `json:"id"`
	Status          Status    `json:"status"`
	TotalPriceCents int64     `json:"total_price_cents"`
	Currency        string    `json:"currency"`
	CreatedAt       time.Time `json:"created_at"`
}

type getOrdersResponse struct {
	Orders []orderBaseInfo `json:"orders"`
	Page   int             `json:"page"`
	Limit  int             `json:"limit"`
}

type orderItemResponse struct {
	ID                  uuid.UUID `json:"id"`
	ProductID           uuid.UUID `json:"product_id"`
	ProductName         string    `json:"product_name"`
	UnitPriceCents      int64     `json:"unit_price_cents"`
	Currency            string    `json:"currency"`
	Quantity            int       `json:"quantity"`
	LineTotalPriceCents int64     `json:"line_total_price_cents"`
	CreatedAt           time.Time `json:"created_at"`
}

type orderResponse struct {
	orderBaseInfo
	Items []orderItemResponse `json:"items"`
}

const (
	defaultOrdersLimit = 20
	maxOrdersLimit     = 100
	defaultOrdersPage  = 1
)

func NewHandler(log *slog.Logger, service *Service) *Handler {
	return &Handler{service: service, log: log}
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	const op = "orders.handler.CreateOrder"

	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		response.Unauthorized(w)
		return
	}

	var reqCreateOrder createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&reqCreateOrder); err != nil {
		response.BadRequest(w)
		return
	}

	order, err := h.service.CreateOrder(r.Context(), userID, reqCreateOrder.ProductIDs)
	if err != nil {
		writeCreateOrderError(w, err, h.log, op)
		return
	}

	res := createOrderResponse{
		ID:              order.ID,
		Status:          order.Status,
		TotalPriceCents: order.TotalPriceCents,
		Currency:        order.Currency,
		CreatedAt:       order.CreatedAt,
	}

	if err := response.JSON(w, http.StatusCreated, res); err != nil {
		h.log.Error("failed to send create order response", slog.String("op", op), slog.String("err", err.Error()))
		response.InternalError(w)
		return
	}
}

func (h *Handler) ListByUserID(w http.ResponseWriter, r *http.Request) {
	const op = "orders.handler.ListByUserID"

	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		response.Unauthorized(w)
		return
	}

	queryParams := r.URL.Query()
	page, ok := parsePagination(queryParams, "page")
	if !ok {
		response.BadRequestMsg(w, "invalid query params")
		return
	}

	if page < defaultOrdersPage {
		page = defaultOrdersPage
	}

	limit, ok := parsePagination(queryParams, "limit")
	if !ok {
		response.BadRequestMsg(w, "invalid query params")
		return
	}

	if limit <= 0 {
		limit = defaultOrdersLimit
	}

	if limit > maxOrdersLimit {
		limit = maxOrdersLimit
	}

	orders, err := h.service.ListByUserID(r.Context(), userID, page, limit)
	if err != nil {
		if errors.Is(err, ErrUserIDIsNil) {
			response.Unauthorized(w)
			return
		}

		h.log.Error("failed to get orders", slog.String("op", op), slog.String("err", err.Error()))
		response.InternalError(w)
		return
	}

	res := toOrdersResponse(orders, page, limit)

	if err := response.JSON(w, http.StatusOK, res); err != nil {
		h.log.Error("failed to send response orders", slog.String("op", op), slog.String("err", err.Error()))
		response.InternalError(w)
		return
	}

}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	const op = "orders.handler.GetByID"

	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		response.Unauthorized(w)
		return
	}

	urlOrderID := chi.URLParam(r, "orderID")
	orderID, err := uuid.Parse(urlOrderID)
	if err != nil {
		response.BadRequestMsg(w, "invalid order id")
		return
	}

	orderDetails, err := h.service.GetByID(r.Context(), userID, orderID)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserIDIsNil):
			response.Unauthorized(w)
			return
		case errors.Is(err, ErrOrderIDIsNil):
			response.BadRequestMsg(w, "invalid order id")
			return
		case errors.Is(err, ErrOrderNotFound):
			response.Error(w, http.StatusNotFound, "order not found")
			return
		default:
			h.log.Error("failed to get order by id", slog.String("op", op), slog.String("err", err.Error()))
			response.InternalError(w)
			return
		}
	}

	res := toOrderResponse(orderDetails)

	if err := response.JSON(w, http.StatusOK, res); err != nil {
		h.log.Error("failed to send order response", slog.String("op", op), slog.String("err", err.Error()))
		response.InternalError(w)
		return
	}

}

func writeCreateOrderError(w http.ResponseWriter, err error, log *slog.Logger, op string) {
	switch {
	case errors.Is(err, ErrUserIDIsNil):
		response.Unauthorized(w)
		return
	case errors.Is(err, ErrProductIDsEmpty):
		response.BadRequestMsg(w, "product_ids must not be empty")
		return
	case errors.Is(err, ErrProductIDIsNil):
		response.BadRequestMsg(w, "product_ids contains an empty UUID")
		return
	case errors.Is(err, ErrDuplicateProductID):
		response.BadRequestMsg(w, "product_ids contains duplicates")
		return
	case errors.Is(err, ErrCartChanged):
		response.Error(
			w,
			http.StatusConflict,
			"cart contents changed, refresh the cart and try again",
		)
		return
	case errors.Is(err, ErrProductInactive):
		response.Error(
			w,
			http.StatusConflict,
			"one or more selected products are unavailable",
		)
		return
	case errors.Is(err, ErrCurrencyMismatch):
		response.Error(
			w,
			http.StatusConflict,
			"selected products have different currencies",
		)
		return
	case errors.Is(err, ErrInsufficientStock):
		response.Error(
			w,
			http.StatusConflict,
			"insufficient stock for one or more selected products",
		)
		return
	case errors.Is(err, ErrInventoryNotFound):
		log.Error(
			"inventory not found while creating order",
			slog.String("op", op),
			slog.String("err", err.Error()),
		)
		response.InternalError(w)
		return
	default:
		log.Error("failed to create order", slog.String("op", op), slog.String("err", err.Error()))
		response.InternalError(w)
		return
	}
}

func parsePagination(urlValues url.Values, key string) (int, bool) {
	str := urlValues.Get(key)
	if str == "" {
		return 0, true
	}

	v, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return 0, false
	}

	if v < 0 {
		return 0, true
	}

	return int(v), true
}

func toOrdersResponse(orders []Order, page, limit int) *getOrdersResponse {
	resOrders := make([]orderBaseInfo, 0, len(orders))
	for _, o := range orders {
		resOrders = append(resOrders, orderBaseInfo{
			ID:              o.ID,
			Status:          o.Status,
			TotalPriceCents: o.TotalPriceCents,
			Currency:        o.Currency,
			CreatedAt:       o.CreatedAt,
			UpdatedAt:       o.UpdatedAt,
		})
	}

	return &getOrdersResponse{
		Orders: resOrders,
		Page:   page,
		Limit:  limit,
	}
}

func toOrderResponse(orderDetails *OrderDetails) *orderResponse {
	orderItems := make([]orderItemResponse, 0, len(orderDetails.Items))
	for _, item := range orderDetails.Items {
		orderItems = append(orderItems, orderItemResponse{
			ID:                  item.ID,
			ProductID:           item.ProductID,
			ProductName:         item.ProductName,
			UnitPriceCents:      item.UnitPriceCents,
			Currency:            item.Currency,
			Quantity:            item.Quantity,
			LineTotalPriceCents: item.LineTotalPriceCents,
			CreatedAt:           item.CreatedAt,
		})
	}

	return &orderResponse{
		orderBaseInfo: orderBaseInfo{
			ID:              orderDetails.ID,
			Status:          orderDetails.Status,
			TotalPriceCents: orderDetails.TotalPriceCents,
			Currency:        orderDetails.Currency,
			CreatedAt:       orderDetails.CreatedAt,
			UpdatedAt:       orderDetails.UpdatedAt,
		},
		Items: orderItems,
	}
}
