package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/MSNZT/orderflow/internal/app/payments"
	"github.com/MSNZT/orderflow/internal/transport/http/httpmw"
	"github.com/MSNZT/orderflow/internal/transport/http/response"
)

type PaymentProcessor interface {
	ProcessSucceededPayment(ctx context.Context, providerPayment payments.ProviderPayment) error
	ProcessCanceledPayment(ctx context.Context, providerPayment payments.ProviderPayment) error
	ProcessWaitingForCapturePayment(ctx context.Context, payment payments.ProviderPayment, now time.Time) error
}

type Handler struct {
	log              *slog.Logger
	resp             *response.Response
	paymentProcessor PaymentProcessor
	paymentProvider  payments.PaymentProvider
}

func NewHandler(
	log *slog.Logger, resp *response.Response,
	paymentProcessor PaymentProcessor, paymentProvider payments.PaymentProvider,
) *Handler {
	return &Handler{
		log:              log,
		resp:             resp,
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

func (h *Handler) YooKassa(w http.ResponseWriter, r *http.Request) error {
	const op = "webhooks.handler.YooKassa"

	var req yookassaWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid webhook payload")
	}

	if !validateYooKassaPaymentEvent(req) {
		return httpmw.NewHTTPError(http.StatusBadRequest, op, "invalid webhook payload")
	}

	switch req.Event {
	case "payment.succeeded":

		return h.processPaymentEvent(
			w,
			r,
			req,
			payments.StatusSucceeded,
			h.paymentProcessor.ProcessSucceededPayment,
		)

	case "payment.canceled":

		return h.processPaymentEvent(
			w,
			r,
			req,
			payments.StatusCanceled,
			h.paymentProcessor.ProcessCanceledPayment,
		)

	case "payment.waiting_for_capture":
		return h.processPaymentEvent(
			w,
			r,
			req,
			payments.StatusWaitingForCapture,
			func(
				ctx context.Context,
				providerPayment payments.ProviderPayment,
			) error {
				return h.paymentProcessor.ProcessWaitingForCapturePayment(
					ctx,
					providerPayment,
					time.Now().UTC(),
				)
			},
		)

	default:
		return nil
	}
}

func (h *Handler) processPaymentEvent(
	w http.ResponseWriter, r *http.Request, req yookassaWebhookRequest, expectedStatus payments.Status,
	process func(ctx context.Context, providerPayment payments.ProviderPayment) error) error {
	const op = "webhooks.handler.processFinalPaymentEvent"

	providerPayment, err := h.paymentProvider.GetPayment(r.Context(), req.Object.ID)
	if err != nil {
		return httpmw.WrapHTTPError(
			http.StatusInternalServerError,
			op,
			"internal server error",
			fmt.Errorf(`
			failed to verify yookassa payment via provider api: provider_payment_id=%s: %w`,
				req.Object.ID, err,
			),
		)
	}

	if providerPayment == nil {
		return httpmw.WrapHTTPError(
			http.StatusInternalServerError,
			op,
			"internal server error",
			fmt.Errorf("provider payment is nil: %v", providerPayment),
		)
	}

	if providerPayment.ID != req.Object.ID {
		return httpmw.WrapHTTPError(
			http.StatusInternalServerError,
			op,
			"internal server error",
			fmt.Errorf("provider payment id mismatch: provider_payment_id=%s, webhook_payment_id=%s",
				providerPayment.ID, req.Object.ID),
		)
	}

	if providerPayment.Status != expectedStatus {
		return httpmw.WrapHTTPError(
			http.StatusInternalServerError,
			op,
			"internal server error",
			fmt.Errorf(`failed to process yookassa payment event: 
				provider_payment_id=%s, 
				webhook_event=%s, 
				provider_status=%s, 
				expected_status=%s`,
				req.Object.ID, req.Event, string(providerPayment.Status), string(expectedStatus),
			))
	}

	if err := process(r.Context(), *providerPayment); err != nil {
		return httpmw.WrapHTTPError(
			http.StatusInternalServerError,
			op,
			"internal server error",
			fmt.Errorf(`failed to process yookassa payment event: 
				provider_payment_id=%s, 
				expected_status=%s: 
				%w`,
				req.Object.ID, string(expectedStatus), err,
			))
	}

	w.WriteHeader(http.StatusOK)
	return nil
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
