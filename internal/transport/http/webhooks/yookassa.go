package webhooks

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/MSNZT/orderflow/internal/app/payments"
	"github.com/MSNZT/orderflow/internal/transport/http/response"
)

type PaymentProcessor interface {
	ProcessSucceededPayment(ctx context.Context, providerPaymentID string) error
	ProcessCanceledPayment(ctx context.Context, providerPaymentID string) error
}

type Handler struct {
	log              *slog.Logger
	paymentProcessor PaymentProcessor
	paymentProvider  payments.PaymentProvider
}

func NewHandler(log *slog.Logger, paymentProcessor PaymentProcessor, paymentProvider payments.PaymentProvider) *Handler {
	return &Handler{
		log:              log,
		paymentProcessor: paymentProcessor,
		paymentProvider:  paymentProvider,
	}
}

type yookassaWebhookRequest struct {
	Type   string                `json:"type"`
	Event  string                `json:"event"`
	Object yookassaPaymentObject `json:"object"`
}

type yookassaPaymentObject struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (h *Handler) YooKassa(w http.ResponseWriter, r *http.Request) {
	const op = "webhooks.handler.YooKassa"

	var req yookassaWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequestMsg(w, "invalid webhook payload")
		return
	}

	if !validateYooKassaPaymentEvent(req) {
		response.BadRequestMsg(w, "invalid webhook payload")
		return
	}

	providerPayment, err := h.paymentProvider.GetPayment(r.Context(), req.Object.ID)
	if err != nil {
		response.BadRequestMsg(w, "failed to get payment")
		return
	}

	if providerPayment.Status != payments.Status(req.Object.Status) {
		response.BadRequestMsg(w, "payment status mismatch")
		return
	}

	switch req.Event {
	case "payment.succeeded":
		if err := h.paymentProcessor.ProcessSucceededPayment(r.Context(), req.Object.ID); err != nil {
			h.log.Error(
				"failed to process yookassa payment succeeded webhook",
				slog.String("op", op),
				slog.String("provider_payment_id", req.Object.ID),
				slog.Any("error", err),
			)
			response.InternalError(w)
			return
		}

		response.NoContent(w)
		return

	case "payment.canceled":
		if err := h.paymentProcessor.ProcessCanceledPayment(r.Context(), req.Object.ID); err != nil {
			h.log.Error(
				"failed to process yookassa payment canceled webhook",
				slog.String("op", op),
				slog.String("provider_payment_id", req.Object.ID),
				slog.Any("error", err),
			)
			response.InternalError(w)
			return
		}

		response.NoContent(w)
		return

	default:
		h.log.Info(
			"ignored yookassa webhook event",
			slog.String("op", op),
			slog.String("event", req.Event),
			slog.String("provider_payment_id", req.Object.ID),
			slog.String("provider_status", req.Object.Status),
		)

		response.NoContent(w)
		return
	}
}

func validateYooKassaPaymentEvent(req yookassaWebhookRequest) bool {
	if req.Type != "notification" {
		return false
	}

	if req.Event == "" {
		return false
	}

	if req.Object.ID == "" {
		return false
	}

	if req.Object.Status == "" {
		return false
	}

	switch req.Event {
	case "payment.succeeded":
		return req.Object.Status == "succeeded"
	case "payment.canceled":
		return req.Object.Status == "canceled"
	case "payment.waiting_for_capture":
		return req.Object.Status == "waiting_for_capture"
	default:
		return true
	}
}
