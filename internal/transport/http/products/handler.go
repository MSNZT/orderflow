package products

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/MSNZT/orderflow/internal/app/products"
	"github.com/MSNZT/orderflow/internal/transport/http/httpmw"
	"github.com/MSNZT/orderflow/internal/transport/http/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Service interface {
	List(ctx context.Context) ([]products.Product, error)
	GetByID(
		ctx context.Context,
		productID uuid.UUID,
	) (*products.Product, error)
	Create(
		ctx context.Context,
		input products.CreateInput,
	) (*products.Product, error)
}

type Handler struct {
	service Service
	resp    *response.Response
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

func NewHandler(resp *response.Response, service Service) *Handler {
	return &Handler{resp: resp, service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) error {
	const op = "products.handler.List"

	products, err := h.service.List(r.Context())
	if err != nil {
		return fmt.Errorf("%s: failed to get product list: %w", op, err)
	}

	respProducts := make([]productResponse, len(products))
	for i := range products {
		respProducts[i] = toProductResponse(&products[i])
	}

	h.resp.JSON(w, http.StatusOK, listResponse{Products: respProducts})

	return nil
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) error {
	const op = "products.handler.GetByID"

	paramId := chi.URLParam(r, "id")
	id, err := uuid.Parse(paramId)
	if err != nil {
		return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid id")
	}

	product, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, products.ErrProductNotFound):
			return httpmw.NewHTTPError(http.StatusNotFound, op, "product not found")
		default:
			return fmt.Errorf("%s: failed to get product by id: %w", op, err)
		}
	}

	if product == nil {
		return fmt.Errorf("%s: business logic violation: service returned nil product", op)
	}

	h.resp.JSON(w, http.StatusOK, toProductResponse(product))
	return nil
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) error {
	const op = "products.handler.Create"

	var req productCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid body")
	}

	input := products.CreateInput{
		Name:            req.Name,
		Description:     req.Description,
		PriceCents:      req.PriceCents,
		Currency:        req.Currency,
		InitialQuantity: req.InitialQuantity,
	}

	product, err := h.service.Create(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, products.ErrProductNameInvalid):
			return httpmw.NewHTTPError(http.StatusUnprocessableEntity, op, products.ErrProductNameInvalid.Error())
		case errors.Is(err, products.ErrProductPriceCentsInvalid):
			return httpmw.NewHTTPError(http.StatusUnprocessableEntity, op, products.ErrProductPriceCentsInvalid.Error())
		case errors.Is(err, products.ErrProductCurrencyInvalid):
			return httpmw.NewHTTPError(http.StatusUnprocessableEntity, op, products.ErrProductCurrencyInvalid.Error())
		case errors.Is(err, products.ErrInitialQuantityInvalid):
			return httpmw.NewHTTPError(http.StatusUnprocessableEntity, op, products.ErrInitialQuantityInvalid.Error())
		case errors.Is(err, products.ErrProductAlreadyExists):
			return httpmw.NewHTTPError(http.StatusConflict, op, products.ErrProductAlreadyExists.Error())
		default:
			return fmt.Errorf("%s: failed to create product: %w", op, err)
		}
	}

	if product == nil {
		return fmt.Errorf("%s: business logic violation: service returned nil product", op)

	}

	h.resp.JSON(w, http.StatusCreated, toProductResponse(product))
	return nil
}

func toProductResponse(p *products.Product) productResponse {
	return productResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		PriceCents:  p.PriceCents,
		Currency:    p.Currency,
	}
}
