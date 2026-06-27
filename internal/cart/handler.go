package cart

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/MSNZT/orderflow/internal/transport/http/authcontext"
	"github.com/MSNZT/orderflow/internal/transport/http/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
	log     *slog.Logger
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

func NewHandler(log *slog.Logger, service *Service) *Handler {
	return &Handler{log: log, service: service}
}

func (h *Handler) GetItems(w http.ResponseWriter, r *http.Request) {
	const op = "cart.handler.GetItems"

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

	if page < defaultCartPage {
		page = defaultCartPage
	}

	limit, ok := parsePagination(queryParams, "limit")
	if !ok {
		response.BadRequestMsg(w, "invalid query params")
		return
	}

	if limit <= 0 {
		limit = defaultCartLimit
	}

	if limit > maxCartLimit {
		limit = maxCartLimit
	}

	input := getItemsInput{
		UserID: userID,
		Page:   page,
		Limit:  limit,
	}

	cart, err := h.service.GetItems(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserIDIsNil):
			response.Unauthorized(w)
			return
		default:
			h.log.Error("failed to get cart items", slog.String("op", op), slog.String("err", err.Error()))
			response.InternalError(w)
			return
		}
	}

	cartItems := make([]cartItemResponse, len(cart.Items))
	for i, item := range cart.Items {
		cartItems[i] = toCartItemResponse(item)
	}

	if err := response.JSON(w, http.StatusOK, listResponse{
		Items:           cartItems,
		TotalPriceCents: cart.TotalPriceCents,
	}); err != nil {
		h.log.Error("failed to send cart response", slog.String("op", op), slog.String("err", err.Error()))
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
}

func (h *Handler) AddItem(w http.ResponseWriter, r *http.Request) {
	const op = "cart.handler.AddItem"

	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		response.Unauthorized(w)
		return
	}

	var req addItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w)
		return
	}

	input := addItemInput{
		UserID:    userID,
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
	}

	if err := h.service.AddItem(r.Context(), input); err != nil {
		switch {
		case errors.Is(err, ErrUserIDIsNil):
			response.Unauthorized(w)
			return
		case errors.Is(err, ErrProductIDIsNil):
			response.Error(w, http.StatusBadRequest, ErrProductIDIsNil.Error())
			return
		case errors.Is(err, ErrQuantityInvalid):
			response.Error(w, http.StatusUnprocessableEntity, ErrQuantityInvalid.Error())
			return
		case errors.Is(err, ErrProductNotAvailable):
			response.Error(w, http.StatusNotFound, ErrProductNotAvailable.Error())
			return
		default:
			h.log.Error("failed to add item to cart items", slog.String("op", op), slog.String("err", err.Error()))
			response.InternalError(w)
			return
		}
	}

	response.NoContent(w)
}

func (h *Handler) UpdateItemQuantity(w http.ResponseWriter, r *http.Request) {
	const op = "cart.handler.UpdateItemQuantity"

	productIDParam := chi.URLParam(r, "productID")
	productID, err := uuid.Parse(productIDParam)
	if err != nil {
		response.BadRequest(w)
		return
	}

	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		response.Unauthorized(w)
		return
	}

	var req updateItemQuantityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w)
		return
	}

	input := updateItemQuantityInput{
		UserID:    userID,
		ProductID: productID,
		Quantity:  req.Quantity,
	}

	if err := h.service.UpdateItemQuantity(r.Context(), input); err != nil {
		switch {
		case errors.Is(err, ErrUserIDIsNil):
			response.Unauthorized(w)
			return
		case errors.Is(err, ErrProductIDIsNil):
			response.BadRequest(w)
			return
		case errors.Is(err, ErrQuantityInvalid):
			response.Error(w, http.StatusUnprocessableEntity, ErrQuantityInvalid.Error())
			return
		case errors.Is(err, ErrCartItemNotFound):
			response.Error(w, http.StatusNotFound, ErrCartItemNotFound.Error())
			return
		default:
			h.log.Error("failed to update item quantity", slog.String("op", op), slog.String("err", err.Error()))
			response.InternalError(w)
			return
		}
	}

	response.NoContent(w)
}

func (h *Handler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	const op = "cart.handler.DeleteItem"

	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		response.Unauthorized(w)
		return
	}

	url := chi.URLParam(r, "productID")
	productID, err := uuid.Parse(url)
	if err != nil {
		response.BadRequest(w)
		return
	}

	if err := h.service.DeleteItem(r.Context(), userID, productID); err != nil {
		switch {
		case errors.Is(err, ErrUserIDIsNil):
			response.Unauthorized(w)
			return
		case errors.Is(err, ErrProductIDIsNil):
			response.BadRequest(w)
			return
		}

		h.log.Error("failed to delete cart item", slog.String("op", op), slog.String("err", err.Error()))
		response.InternalError(w)
		return
	}

	response.NoContent(w)
}

func (h *Handler) ClearItems(w http.ResponseWriter, r *http.Request) {
	const op = "cart.handler.ClearItems"

	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		response.Unauthorized(w)
		return
	}

	if err := h.service.ClearItems(r.Context(), userID); err != nil {
		if errors.Is(err, ErrUserIDIsNil) {
			response.Unauthorized(w)
			return
		}
		h.log.Error("failed to clear cart items", slog.String("op", op), slog.String("err", err.Error()))
		response.InternalError(w)
		return
	}

	response.NoContent(w)
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

func toCartItemResponse(item CartItem) cartItemResponse {
	return cartItemResponse{
		ProductID:           item.ProductID,
		Name:                item.Name,
		PriceCents:          item.PriceCents,
		Quantity:            item.Quantity,
		LineTotalPriceCents: item.LineTotalPriceCents,
	}
}
