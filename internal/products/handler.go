package products

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/MSNZT/orderflow/internal/transport/http/response"
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
	Name            string  `json:"name"`
	Description     *string `json:"description"`
	PriceCents      int64   `json:"price_cents"`
	Currency        string  `json:"currency"`
	InitialQuantity int32   `json:"initial_quantity"`
}

func NewHandler(log *slog.Logger, service *Service) *Handler {
	return &Handler{log: log, service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	const op = "products.handler.List"

	products, err := h.service.List(r.Context())
	if err != nil {
		h.log.Error("failed to get product list", slog.String("op", op), slog.String("error", err.Error()))
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	respProducts := make([]productResponse, len(products))
	for i, p := range products {
		respProducts[i] = toProductResponse(p)
	}

	if err := response.JSON(w, http.StatusOK, listResponse{Products: respProducts}); err != nil {
		h.log.Error("failed to send json response", slog.String("op", op), slog.String("error", err.Error()))
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	const op = "products.handler.GetByID"

	paramId := chi.URLParam(r, "id")
	id, err := uuid.Parse(paramId)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	product, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, ErrProductNotFound):
			response.Error(w, http.StatusNotFound, "product not found")
			return
		default:
			h.log.Error("failed to get product by id", slog.String("op", op), slog.String("error", err.Error()))
			response.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	if err := response.JSON(w, http.StatusOK, toProductResponse(*product)); err != nil {
		h.log.Error("failed to send json response", slog.String("op", op), slog.String("error", err.Error()))
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	const op = "products.handler.Create"

	var req productCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid body")
		return
	}

	input := createInput{
		Name:            req.Name,
		Description:     req.Description,
		PriceCents:      req.PriceCents,
		Currency:        req.Currency,
		InitialQuantity: req.InitialQuantity,
	}

	product, err := h.service.Create(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, ErrProductNameInvalid):
			response.Error(w, http.StatusUnprocessableEntity, ErrProductNameInvalid.Error())
			return
		case errors.Is(err, ErrProductPriceCentsInvalid):
			response.Error(w, http.StatusUnprocessableEntity, ErrProductPriceCentsInvalid.Error())
			return
		case errors.Is(err, ErrProductCurrencyInvalid):
			response.Error(w, http.StatusUnprocessableEntity, ErrProductCurrencyInvalid.Error())
			return
		case errors.Is(err, ErrInitialQuantityInvalid):
			response.Error(w, http.StatusUnprocessableEntity, ErrInitialQuantityInvalid.Error())
			return
		case errors.Is(err, ErrProductAlreadyExists):
			response.Error(w, http.StatusConflict, ErrProductAlreadyExists.Error())
			return
		default:
			h.log.Error("failed to create product", slog.String("op", op), slog.String("err", err.Error()))
			response.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	if err = response.JSON(w, http.StatusCreated, toProductResponse(*product)); err != nil {
		h.log.Error("failed to send product response", slog.String("op", op), slog.String("err", err.Error()))
		response.Error(w, http.StatusInternalServerError, "internal server error")
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
