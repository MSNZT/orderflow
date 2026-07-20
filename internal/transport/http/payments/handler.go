package payments

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/MSNZT/orderflow/internal/app/orders"
	paymentsapp "github.com/MSNZT/orderflow/internal/app/payments"
	"github.com/MSNZT/orderflow/internal/transport/http/authcontext"
	"github.com/MSNZT/orderflow/internal/transport/http/httpmw"
	"github.com/MSNZT/orderflow/internal/transport/http/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Service interface {
	CreatePayment(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (*paymentsapp.Payment, error)
}

type Handler struct {
	service Service
	resp    *response.Response
}

func NewHandler(resp *response.Response, service Service) *Handler {
	return &Handler{resp: resp, service: service}
}

func (h *Handler) CreatePayment(w http.ResponseWriter, r *http.Request) error {
	const op = "payments.handler.CreatePayment"

	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")
	}

	idStr := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if idStr == "" {
		return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid order id")
	}

	orderID, err := uuid.Parse(idStr)
	if err != nil {
		return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid order id")
	}

	payment, err := h.service.CreatePayment(r.Context(), userID, orderID)
	if err != nil {
		return h.writeCreatePaymentError(err, op)
	}

	res := createPaymentResponse{
		ID:              payment.ID,
		OrderID:         payment.OrderID,
		Status:          payment.Status,
		AmountCents:     payment.AmountCents,
		Currency:        payment.Currency,
		ConfirmationURL: payment.ConfirmationURL,
	}

	h.resp.JSON(w, http.StatusOK, res)
	return nil
}

func (h *Handler) writeCreatePaymentError(err error, op string) error {
	switch {
	case errors.Is(err, paymentsapp.ErrUserIDIsNil):
		return httpmw.NewHTTPError(http.StatusUnauthorized, op, "unauthorized")

	case errors.Is(err, paymentsapp.ErrOrderIDIsNil):
		return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid order id")

	case errors.Is(err, orders.ErrOrderNotFound):
		return httpmw.NewHTTPError(http.StatusNotFound, op, "order not found")

	case errors.Is(err, paymentsapp.ErrOrderNotPayable):
		return httpmw.NewHTTPError(http.StatusConflict, op, "order is not payable")

	case errors.Is(err, paymentsapp.ErrOrderExpired):
		return httpmw.NewHTTPError(http.StatusConflict, op, "order has expired")

	case errors.Is(err, paymentsapp.ErrPaymentStateConflict):
		return httpmw.NewHTTPError(http.StatusConflict, op, "payment state conflict")

	case errors.Is(err, paymentsapp.ErrProviderRejected):
		return httpmw.WrapHTTPError(http.StatusBadGateway, op, "payment provider rejected request", err)

	case errors.Is(err, paymentsapp.ErrProviderFailure):
		return httpmw.WrapHTTPError(http.StatusServiceUnavailable, op, "payment provider is temporarily unavailable", err)
	default:
		return fmt.Errorf("%s: create payment failed: %w", op, err)
	}
}
