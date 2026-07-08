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
	ProcessSucceededPayment(ctx context.Context, providerPayment payments.ProviderPayment) error
	ProcessCanceledPayment(ctx context.Context, providerPayment payments.ProviderPayment) error
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

	switch req.Event {
	case "payment.succeeded":
		h.processFinalPaymentEvent(
			w,
			r,
			req,
			payments.StatusSucceeded,
			h.paymentProcessor.ProcessSucceededPayment,
		)
		return
	case "payment.canceled":
		h.processFinalPaymentEvent(
			w,
			r,
			req,
			payments.StatusCanceled,
			h.paymentProcessor.ProcessCanceledPayment,
		)
		return
	case "payment.waiting_for_capture":
		h.log.Info(
			"ignored yookassa waiting_for_capture webhook",
			slog.String("provider_payment_id", req.Object.ID),
			slog.String("op", op),
		)
		response.NoContent(w)
		return

	default:
		h.log.Info(
			"ignored yookassa webhook event",
			slog.String("event", req.Event),
			slog.String("provider_payment_id", req.Object.ID),
			slog.String("op", op),
		)
		response.NoContent(w)
		return

	}
}

func (h *Handler) processFinalPaymentEvent(
	w http.ResponseWriter, r *http.Request, req yookassaWebhookRequest, expectedStatus payments.Status,
	process func(ctx context.Context, providerPayment payments.ProviderPayment) error) {
	const op = "webhooks.handler.processFinalPaymentEvent"

	providerPayment, err := h.paymentProvider.GetPayment(r.Context(), req.Object.ID)
	if err != nil {
		h.log.Error(
			"failed to verify yookassa payment via provider api",
			slog.String("op", op),
			slog.String("provider_payment_id", req.Object.ID),
			slog.Any("error", err),
		)
		response.InternalError(w)
		return
	}

	if providerPayment == nil {
		h.log.Error(
			"provider payment is nil",
			slog.String("op", op),
		)
		response.InternalError(w)
		return
	}

	if providerPayment.ID != req.Object.ID {
		h.log.Error(
			"provider payment id mismatch",
			slog.String("op", op),
			slog.String("webhook_payment_id", req.Object.ID),
			slog.String("provider_payment_id", providerPayment.ID),
		)
		response.InternalError(w)
		return
	}

	if providerPayment.Status != expectedStatus {
		h.log.Warn(
			"provider payment status mismatch",
			slog.String("op", op),
			slog.String("provider_payment_id", req.Object.ID),
			slog.String("webhook_event", req.Event),
			slog.String("provider_status", string(providerPayment.Status)),
			slog.String("expected_status", string(expectedStatus)),
		)
		response.InternalError(w)
		return
	}

	if err := process(r.Context(), *providerPayment); err != nil {
		h.log.Error(
			"failed to process yookassa payment event",
			slog.String("op", op),
			slog.String("provider_payment_id", req.Object.ID),
			slog.String("expected_status", string(expectedStatus)),
			slog.Any("error", err),
		)
		response.InternalError(w)
		return
	}

	response.NoContent(w)
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
