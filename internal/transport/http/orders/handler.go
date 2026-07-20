package orders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	ordersapp "github.com/MSNZT/orderflow/internal/app/orders"
	"github.com/MSNZT/orderflow/internal/transport/http/authcontext"
	"github.com/MSNZT/orderflow/internal/transport/http/httpmw"
	"github.com/MSNZT/orderflow/internal/transport/http/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Service interface {
	ListByUserID(ctx context.Context, userID uuid.UUID, page int, limit int) ([]ordersapp.Order, error)
	GetByID(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (*ordersapp.OrderDetails, error)
	CreateOrder(ctx context.Context, userID uuid.UUID, productIDs []uuid.UUID) (*ordersapp.Order, error)
}

type Handler struct {
	service Service
	resp    *response.Response
}

type createOrderRequest struct {
	ProductIDs []uuid.UUID `json:"product_ids"`
}

type orderBaseInfo struct {
	ID              uuid.UUID        `json:"id"`
	Status          ordersapp.Status `json:"status"`
	TotalPriceCents int64            `json:"total_price_cents"`
	Currency        string           `json:"currency"`
	ExpiresAt       time.Time        `json:"expires_at"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

type createOrderResponse struct {
	ID              uuid.UUID        `json:"id"`
	Status          ordersapp.Status `json:"status"`
	TotalPriceCents int64            `json:"total_price_cents"`
	Currency        string           `json:"currency"`
	ExpiresAt       time.Time        `json:"expires_at"`
	CreatedAt       time.Time        `json:"created_at"`
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

func NewHandler(resp *response.Response, service Service) *Handler {
	return &Handler{service: service, resp: resp}
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) error {
	const op = "orders.handler.CreateOrder"

	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")
	}

	var reqCreateOrder createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&reqCreateOrder); err != nil {
		return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid request body")
	}

	order, err := h.service.CreateOrder(r.Context(), userID, reqCreateOrder.ProductIDs)
	if err != nil {
		return h.writeCreateOrderError(err, op)
	}

	res := createOrderResponse{
		ID:              order.ID,
		Status:          order.Status,
		TotalPriceCents: order.TotalPriceCents,
		Currency:        order.Currency,
		ExpiresAt:       order.ExpiresAt,
		CreatedAt:       order.CreatedAt,
	}

	h.resp.JSON(w, http.StatusCreated, res)
	return nil
}

func (h *Handler) ListByUserID(w http.ResponseWriter, r *http.Request) error {
	const op = "orders.handler.ListByUserID"

	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")
	}

	queryParams := r.URL.Query()
	page, ok := parsePagination(queryParams, "page")
	if !ok {
		return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid query params")
	}

	if page < defaultOrdersPage {
		page = defaultOrdersPage
	}

	limit, ok := parsePagination(queryParams, "limit")
	if !ok {
		return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid query params")
	}

	if limit <= 0 {
		limit = defaultOrdersLimit
	}

	if limit > maxOrdersLimit {
		limit = maxOrdersLimit
	}

	orders, err := h.service.ListByUserID(r.Context(), userID, page, limit)
	if err != nil {
		if errors.Is(err, ordersapp.ErrUserIDIsNil) {
			return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")
		}

		return fmt.Errorf("%s: failed to get orders: %w", op, err)
	}

	res := toOrdersResponse(orders, page, limit)

	h.resp.JSON(w, http.StatusOK, res)
	return nil
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) error {
	const op = "orders.handler.GetByID"

	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")
	}

	urlOrderID := chi.URLParam(r, "orderID")
	orderID, err := uuid.Parse(urlOrderID)
	if err != nil {
		return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid order id")
	}

	orderDetails, err := h.service.GetByID(r.Context(), userID, orderID)
	if err != nil {
		switch {
		case errors.Is(err, ordersapp.ErrUserIDIsNil):
			return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")
		case errors.Is(err, ordersapp.ErrOrderIDIsNil):
			return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid order id")
		case errors.Is(err, ordersapp.ErrOrderNotFound):
			return httpmw.NewHTTPError(http.StatusNotFound, op, "order not found")
		default:
			return fmt.Errorf("%s: failed to get order by id: %w", op, err)
		}
	}

	res := toOrderResponse(orderDetails)

	h.resp.JSON(w, http.StatusOK, res)
	return nil
}

func (h *Handler) writeCreateOrderError(err error, op string) error {
	switch {
	case errors.Is(err, ordersapp.ErrUserIDIsNil):
		return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")
	case errors.Is(err, ordersapp.ErrProductIDsEmpty):
		return httpmw.NewHTTPError(http.StatusBadRequest, op, "product_ids must not be empty")
	case errors.Is(err, ordersapp.ErrProductIDIsNil):
		return httpmw.NewHTTPError(http.StatusBadRequest, op, "product_ids contains an empty UUID")
	case errors.Is(err, ordersapp.ErrDuplicateProductID):
		return httpmw.NewHTTPError(http.StatusBadRequest, op, "product_ids contains duplicates")
	case errors.Is(err, ordersapp.ErrCartChanged):
		return httpmw.NewHTTPError(
			http.StatusConflict, op,
			"cart contents changed, refresh the cart and try again")
	case errors.Is(err, ordersapp.ErrProductInactive):
		return httpmw.NewHTTPError(
			http.StatusConflict, op,
			"one or more selected products are unavailable")
	case errors.Is(err, ordersapp.ErrCurrencyMismatch):
		return httpmw.NewHTTPError(
			http.StatusConflict, op,
			"selected products have different currencies")
	case errors.Is(err, ordersapp.ErrInsufficientStock):
		return httpmw.NewHTTPError(
			http.StatusConflict, op,
			"insufficient stock for one or more selected products")
	case errors.Is(err, ordersapp.ErrInventoryNotFound):
		return fmt.Errorf("%s: inventory not found while creating order: %w", op, err)
	default:
		return fmt.Errorf("%s: failed to create order: %w", op, err)
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

func toOrdersResponse(orders []ordersapp.Order, page, limit int) *getOrdersResponse {
	resOrders := make([]orderBaseInfo, 0, len(orders))
	for _, o := range orders {
		resOrders = append(resOrders, orderBaseInfo{
			ID:              o.ID,
			Status:          o.Status,
			TotalPriceCents: o.TotalPriceCents,
			Currency:        o.Currency,
			ExpiresAt:       o.ExpiresAt,
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

func toOrderResponse(orderDetails *ordersapp.OrderDetails) *orderResponse {
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
			ExpiresAt:       orderDetails.ExpiresAt,
			CreatedAt:       orderDetails.CreatedAt,
			UpdatedAt:       orderDetails.UpdatedAt,
		},
		Items: orderItems,
	}
}
