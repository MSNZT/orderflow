package cart

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	cartapp "github.com/MSNZT/orderflow/internal/app/cart"
	"github.com/MSNZT/orderflow/internal/transport/http/authcontext"
	"github.com/MSNZT/orderflow/internal/transport/http/httpmw"
	"github.com/MSNZT/orderflow/internal/transport/http/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Service interface {
	GetItems(ctx context.Context, input cartapp.GetItemsInput) (*cartapp.Cart, error)
	AddItem(ctx context.Context, input cartapp.AddItemInput) error
	UpdateItemQuantity(ctx context.Context, input cartapp.UpdateItemQuantityInput) error
	DeleteItem(ctx context.Context, userID uuid.UUID, productID uuid.UUID) error
	ClearItems(ctx context.Context, userID uuid.UUID) error
}

type Handler struct {
	service Service
	resp    *response.Response
}

type listResponse struct {
	Items           []cartItemResponse `json:"items"`
	TotalPriceCents int64              `json:"total_price_cents"`
}

type cartItemResponse struct {
	ProductID           uuid.UUID `json:"product_id"`
	Name                string    `json:"name"`
	PriceCents          int64     `json:"price_cents"`
	Quantity            int32     `json:"quantity"`
	LineTotalPriceCents int64     `json:"line_total_price_cents"`
}

type addItemRequest struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int32     `json:"quantity"`
}

type updateItemQuantityRequest struct {
	Quantity int32 `json:"quantity"`
}

const (
	defaultCartLimit = 20
	maxCartLimit     = 100
	defaultCartPage  = 1
)

func NewHandler(resp *response.Response, service Service) *Handler {
	return &Handler{resp: resp, service: service}
}

func (h *Handler) GetItems(w http.ResponseWriter, r *http.Request) error {
	const op = "cart.handler.GetItems"

	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")
	}

	queryParams := r.URL.Query()
	page, ok := parsePagination(queryParams, "page")
	if !ok {
		return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid query params")
	}

	if page < defaultCartPage {
		page = defaultCartPage
	}

	limit, ok := parsePagination(queryParams, "limit")
	if !ok {
		return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid query params")
	}

	if limit <= 0 {
		limit = defaultCartLimit
	}

	if limit > maxCartLimit {
		limit = maxCartLimit
	}

	input := cartapp.GetItemsInput{
		UserID: userID,
		Page:   page,
		Limit:  limit,
	}

	cart, err := h.service.GetItems(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, cartapp.ErrUserIDIsNil):
			return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")
		default:
			return fmt.Errorf("%s: failed to get cart items: %w", op, err)
		}
	}

	cartItems := make([]cartItemResponse, len(cart.Items))
	for i, item := range cart.Items {
		cartItems[i] = toCartItemResponse(item)
	}

	h.resp.JSON(w, http.StatusOK, listResponse{
		Items:           cartItems,
		TotalPriceCents: cart.TotalPriceCents,
	})

	return nil
}

func (h *Handler) AddItem(w http.ResponseWriter, r *http.Request) error {
	const op = "cart.handler.AddItem"

	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")
	}

	var req addItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid request body")
	}

	input := cartapp.AddItemInput{
		UserID:    userID,
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
	}

	if err := h.service.AddItem(r.Context(), input); err != nil {
		switch {
		case errors.Is(err, cartapp.ErrUserIDIsNil):
			return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")
		case errors.Is(err, cartapp.ErrProductIDIsNil):
			return httpmw.NewHTTPError(http.StatusBadRequest, op, cartapp.ErrProductIDIsNil.Error())
		case errors.Is(err, cartapp.ErrQuantityInvalid):
			return httpmw.NewHTTPError(http.StatusUnprocessableEntity, op, cartapp.ErrQuantityInvalid.Error())
		case errors.Is(err, cartapp.ErrProductNotAvailable):
			return httpmw.NewHTTPError(http.StatusNotFound, op, cartapp.ErrProductNotAvailable.Error())
		default:
			return fmt.Errorf("%s: failed to add item to cart items: %w", op, err)
		}
	}

	h.resp.NoContent(w)
	return nil
}

func (h *Handler) UpdateItemQuantity(w http.ResponseWriter, r *http.Request) error {
	const op = "cart.handler.UpdateItemQuantity"

	productIDParam := chi.URLParam(r, "productID")
	productID, err := uuid.Parse(productIDParam)
	if err != nil {
		return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid product id")
	}

	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")
	}

	var req updateItemQuantityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid request body")
	}

	input := cartapp.UpdateItemQuantityInput{
		UserID:    userID,
		ProductID: productID,
		Quantity:  req.Quantity,
	}

	if err := h.service.UpdateItemQuantity(r.Context(), input); err != nil {
		switch {
		case errors.Is(err, cartapp.ErrUserIDIsNil):
			return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")
		case errors.Is(err, cartapp.ErrProductIDIsNil):
			return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid product id")
		case errors.Is(err, cartapp.ErrQuantityInvalid):
			return httpmw.NewHTTPError(http.StatusUnprocessableEntity, op, cartapp.ErrQuantityInvalid.Error())
		case errors.Is(err, cartapp.ErrCartItemNotFound):
			return httpmw.NewHTTPError(http.StatusNotFound, op, cartapp.ErrCartItemNotFound.Error())
		default:
			return fmt.Errorf("%s: failed to update item quantity: %w", op, err)
		}
	}

	h.resp.NoContent(w)

	return nil
}

func (h *Handler) DeleteItem(w http.ResponseWriter, r *http.Request) error {
	const op = "cart.handler.DeleteItem"

	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")
	}

	url := chi.URLParam(r, "productID")
	productID, err := uuid.Parse(url)
	if err != nil {
		return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid product id")
	}

	if err := h.service.DeleteItem(r.Context(), userID, productID); err != nil {
		switch {
		case errors.Is(err, cartapp.ErrUserIDIsNil):
			return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")
		case errors.Is(err, cartapp.ErrProductIDIsNil):
			return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid product id")
		default:
			return fmt.Errorf("%s: failed to delete cart item: %w", op, err)
		}
	}

	h.resp.NoContent(w)
	return nil
}

func (h *Handler) ClearItems(w http.ResponseWriter, r *http.Request) error {
	const op = "cart.handler.ClearItems"

	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")
	}

	if err := h.service.ClearItems(r.Context(), userID); err != nil {
		if errors.Is(err, cartapp.ErrUserIDIsNil) {
			return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")
		}
		return fmt.Errorf("%s: failed to clear cart items: %w", op, err)
	}

	h.resp.NoContent(w)
	return nil
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

func toCartItemResponse(item cartapp.CartItem) cartItemResponse {
	return cartItemResponse{
		ProductID:           item.ProductID,
		Name:                item.Name,
		PriceCents:          item.PriceCents,
		Quantity:            item.Quantity,
		LineTotalPriceCents: item.LineTotalPriceCents,
	}
}
