package products

import (
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

type listResponse struct {
	Products []Product `json:"products"`
}

type productResponse struct {
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
		httpresponse.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if err := httpresponse.JSON(w, http.StatusOK, listResponse{Products: products}); err != nil {
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

	if err := httpresponse.JSON(w, http.StatusOK, productResponse{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		PriceCents:  product.PriceCents,
		Currency:    product.Currency,
	}); err != nil {
		h.log.Error("failed to send json response", slog.String("op", op), slog.String("error", err.Error()))
		httpresponse.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
}
