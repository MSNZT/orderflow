package products

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/MSNZT/orderflow/internal/httpresponse"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
	log     *slog.Logger
}

type productResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	PriceCents  int64     `json:"price_cents"`
	Currency    string    `json:"currency"`
}

type listResponse struct {
	Products []productResponse `json:"products"`
}

type productCreateRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	PriceCents  int64   `json:"price_cents"`
	Currency    *string `json:"currency"`
	Quantity    int32   `json:"quantity"`
	IsActive    *bool   `json:"is_active"`
}

type productCreateResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	PriceCents  int64     `json:"price_cents"`
	Currency    string    `json:"currency"`
}

func NewHandler(log *slog.Logger, service *Service) *Handler {
	return &Handler{log: log, service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	const op = "products.handler.List"

	products, err := h.service.List(r.Context())
	if err != nil {
		h.log.Error("failed to get product list", slog.String("op", op), slog.String("error", err.Error()))
		httpresponse.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	respProducts := make([]productResponse, len(products))
	for i, p := range products {
		respProducts[i] = toProductResponse(p)
	}

	if err := httpresponse.JSON(w, http.StatusOK, listResponse{Products: respProducts}); err != nil {
		h.log.Error("failed to send json response", slog.String("op", op), slog.String("error", err.Error()))
		httpresponse.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	const op = "products.handler.GetByID"

	paramId := chi.URLParam(r, "id")
	id, err := uuid.Parse(paramId)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	product, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, ErrProductNotFound):
			httpresponse.Error(w, http.StatusNotFound, "product not found")
			return
		default:
			h.log.Error("failed to get product by id", slog.String("op", op), slog.String("error", err.Error()))
			httpresponse.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	if err := httpresponse.JSON(w, http.StatusOK, toProductResponse(*product)); err != nil {
		h.log.Error("failed to send json response", slog.String("op", op), slog.String("error", err.Error()))
		httpresponse.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	const op = "products.handler.Create"

	var req productCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "invalid body")
		return
	}

	id, err := uuid.NewV7()
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	var currency string
	if req.Currency != nil {
		currency = *req.Currency
	} else {
		currency = "RUB"
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	p := Product{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		PriceCents:  req.PriceCents,
		Currency:    currency,
		IsActive:    isActive,
	}

	product, err := h.service.Create(r.Context(), &p, req.Quantity)
	if err != nil {
		h.log.Error("failed to create product", slog.String("op", op), slog.String("err", err.Error()))
		httpresponse.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	res := productCreateResponse{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		PriceCents:  product.PriceCents,
		Currency:    product.Currency,
	}

	if err = httpresponse.JSON(w, http.StatusCreated, res); err != nil {
		h.log.Error("failed to send product response", slog.String("op", op), slog.String("err", err.Error()))
		httpresponse.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
}

func toProductResponse(p Product) productResponse {
	return productResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		PriceCents:  p.PriceCents,
		Currency:    p.Currency,
	}
}
