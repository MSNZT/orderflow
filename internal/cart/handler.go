package cart

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/MSNZT/orderflow/internal/authcontext"
	"github.com/MSNZT/orderflow/internal/httpresponse"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
	log     *slog.Logger
}

type listResponse struct {
	Items           []CartItem `json:"items"`
	TotalPriceCents int64      `json:"total_price_cents"`
}

type addItemRequest struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int32     `json:"quantity"`
}

func NewHandler(log *slog.Logger, service *Service) *Handler {
	return &Handler{log: log, service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	const op = "cart.handler.List"

	userId, ok := authcontext.UserID(r.Context())
	if !ok {
		httpresponse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	queryParams := r.URL.Query()
	page, ok := parsePagination(queryParams, "page")
	if !ok {
		httpresponse.Error(w, http.StatusBadRequest, "invalid query params")
		return
	}
	limit, ok := parsePagination(queryParams, "limit")
	if !ok {
		if !ok {
			httpresponse.Error(w, http.StatusBadRequest, "invalid query params")
			return
		}
	}

	input := listInput{
		UserID: userId,
		Page:   page,
		Limit:  limit,
	}

	cart, err := h.service.List(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserIDIsNil):
			httpresponse.Error(w, http.StatusUnauthorized, "unauthorized")
			return
		default:
			h.log.Error("failed to get cart items", slog.String("op", op), slog.String("err", err.Error()))
			httpresponse.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	if err := httpresponse.JSON(w, http.StatusOK, listResponse{
		Items:           cart.Items,
		TotalPriceCents: cart.TotalPriceCents,
	}); err != nil {
		h.log.Error("failed to send cart response", slog.String("op", op), slog.String("err", err.Error()))
		httpresponse.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
}

func (h *Handler) AddItem(w http.ResponseWriter, r *http.Request) {
	const op = "cart.handler.AddItem"

	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		httpresponse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req addItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "bad request")
		return
	}

	input := addItemInput{
		UserID:    userID,
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
	}

	if err := h.service.AddItem(r.Context(), input); err != nil {
		h.log.Error("failed to add item to cart items", slog.String("op", op), slog.String("err", err.Error()))
		httpresponse.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	httpresponse.NoContent(w)
}

func parsePagination(urlValues url.Values, key string) (int32, bool) {
	str := urlValues.Get(key)
	if str == "" {
		return int32(0), true
	}

	v, err := strconv.ParseInt(str, 10, 32)
	if err != nil || v < 0 {
		return int32(0), false
	}
	return int32(v), true
}
