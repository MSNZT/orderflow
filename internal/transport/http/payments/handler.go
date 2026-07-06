package payments

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/MSNZT/orderflow/internal/app/orders"
	paymentsapp "github.com/MSNZT/orderflow/internal/app/payments"
	"github.com/MSNZT/orderflow/internal/transport/http/authcontext"
	"github.com/MSNZT/orderflow/internal/transport/http/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Service interface {
	CreatePayment(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (*paymentsapp.Payment, error)
}

type Handler struct {
	service Service
	log     *slog.Logger
}

func NewHandler(log *slog.Logger, service Service) *Handler {
	return &Handler{log: log, service: service}
}

func (h *Handler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	const op = "payments.handler.CreatePayment"

	userID, ok := authcontext.UserID(r.Context())
	if !ok {
		response.Unauthorized(w)
		return
	}

	idStr := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if idStr == "" {
		response.BadRequestMsg(w, "invalid order id")
		return
	}

	orderID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequestMsg(w, "invalid order id")
		return
	}

	payment, err := h.service.CreatePayment(r.Context(), userID, orderID)
	if err != nil {
		writeCreatePaymentError(w, err, h.log, op)
		return
	}
	res := createPaymentResponse{
		ID:              payment.ID,
		OrderID:         payment.OrderID,
		Status:          payment.Status,
		AmountCents:     payment.AmountCents,
		Currency:        payment.Currency,
		ConfirmationURL: payment.ConfirmationURL,
	}
	if err := response.JSON(w, http.StatusOK, res); err != nil {
		response.InternalError(w)
		h.log.Error("failed to send payment response", slog.String("op", op), slog.String("error", err.Error()))
		return
	}
}

func writeCreatePaymentError(w http.ResponseWriter, err error, log *slog.Logger, op string) {
	switch {
	case errors.Is(err, paymentsapp.ErrUserIDIsNil):
		response.Unauthorized(w)
	case errors.Is(err, paymentsapp.ErrOrderIDIsNil):
		response.BadRequestMsg(w, "invalid order id")
	case errors.Is(err, orders.ErrOrderNotFound):
		response.Error(w, http.StatusNotFound, "order not found")
	case errors.Is(err, paymentsapp.ErrOrderNotPayable):
		response.Error(w, http.StatusConflict, "order is not payable")
	case errors.Is(err, paymentsapp.ErrOrderExpired):
		response.Error(w, http.StatusConflict, "order has expired")
	case errors.Is(err, paymentsapp.ErrPaymentStateConflict):
		response.Error(w, http.StatusConflict, "payment state conflict")
	case errors.Is(err, paymentsapp.ErrProviderRejected):
		log.Error("payment provider rejected request", slog.String("op", op), slog.Any("error", err))
		response.Error(w, http.StatusBadGateway, "payment provider rejected request")
	case errors.Is(err, paymentsapp.ErrProviderFailure):
		log.Error("payment provider failure", slog.String("op", op), slog.Any("error", err))
		response.Error(w, http.StatusServiceUnavailable, "payment provider is temporarily unavailable")
	default:
		log.Error("create payment failed", slog.String("op", op), slog.Any("error", err))
		response.Error(w, http.StatusInternalServerError, "internal server error")
	}
}
