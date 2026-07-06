package yookassa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

type YookassaClientConfig struct {
	APIURL     string
	ShopID     string
	SecretKey  string
	ReturnURL  string
	HTTPClient *http.Client
}

type Client struct {
	apiURL     string
	shopID     string
	secretKey  string
	returnURL  string
	httpClient *http.Client
}

func NewClient(cfg YookassaClientConfig) (*Client, error) {
	apiURL := strings.TrimRight(strings.TrimSpace(cfg.APIURL), "/")
	shopID := strings.TrimSpace(cfg.ShopID)
	secretKey := strings.TrimSpace(cfg.SecretKey)
	returnURL := strings.TrimSpace(cfg.ReturnURL)

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &Client{
		apiURL:     apiURL,
		shopID:     shopID,
		secretKey:  secretKey,
		returnURL:  returnURL,
		httpClient: cfg.HTTPClient,
	}, nil
}

func (c *YookassaClientConfig) validate() error {
	apiURL := strings.TrimRight(strings.TrimSpace(c.APIURL), "/")
	shopID := strings.TrimSpace(c.ShopID)
	secretKey := strings.TrimSpace(c.SecretKey)
	returnURL := strings.TrimSpace(c.ReturnURL)

	if c.HTTPClient == nil {
		return fmt.Errorf("yookassa client: nil HTTP client: %w", ErrInvalidArgument)
	}

	parsedAPIURL, err := url.ParseRequestURI(apiURL)
	if err != nil {
		return fmt.Errorf("yookassa client: invalid api URL: %w", ErrInvalidArgument)
	}
	if parsedAPIURL.Scheme != "http" && parsedAPIURL.Scheme != "https" {
		return fmt.Errorf("yookassa client: unsupported api URL scheme: %w", ErrInvalidArgument)
	}
	if parsedAPIURL.Host == "" {
		return fmt.Errorf("yookassa client: empty api URL host: %w", ErrInvalidArgument)
	}

	if shopID == "" {
		return fmt.Errorf("yookassa client: empty shop id: %w", ErrInvalidArgument)
	}

	if secretKey == "" {
		return fmt.Errorf("yookassa client: empty secret key: %w", ErrInvalidArgument)
	}

	parsedReturnURL, err := url.ParseRequestURI(returnURL)
	if err != nil {
		return fmt.Errorf("yookassa client: invalid return URL: %w", ErrInvalidArgument)
	}

	if parsedReturnURL.Scheme != "http" && parsedReturnURL.Scheme != "https" {
		return fmt.Errorf("yookassa client: unsupported return URL scheme: %w", ErrInvalidArgument)
	}

	if parsedReturnURL.Host == "" {
		return fmt.Errorf("yookassa client: empty return URL host: %w", ErrInvalidArgument)
	}

	return nil
}

func (c *Client) CreatePayment(ctx context.Context, params CreatePaymentParams) (*Payment, error) {
	const op = "yookassa.client.CreatePayment"

	if err := params.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	paymentReq := c.buildCreatePaymentRequest(params)

	jsonData, err := json.Marshal(paymentReq)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	fullURL := strings.TrimRight(c.apiURL, "/") + "/payments"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("%s: create request: %w", op, err)
	}

	req.Header.Set("Idempotence-Key", params.IdempotencyKey.String())
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.shopID, c.secretKey)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w: %v", op, ErrResultUnknown, err)
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		apiError := APIError{
			StatusCode: res.StatusCode,
			Cause:      mapStatusCode(res.StatusCode),
		}

		var errResponse apiErrorResponse
		if err := json.NewDecoder(res.Body).Decode(&errResponse); err == nil {
			apiError.ID = errResponse.ID
			apiError.Code = errResponse.Code
			apiError.Description = errResponse.Description
			apiError.Parameter = errResponse.Parameter
		}

		return nil, fmt.Errorf("%s: %w", op, &apiError)
	}

	var payment Payment
	if err := json.NewDecoder(res.Body).Decode(&payment); err != nil {
		return nil, fmt.Errorf("%s: decode payment response: %w: %v", op, ErrResultUnknown, err)
	}

	if err := validateCreatePaymentResponse(&payment, paymentReq); err != nil {
		return nil, fmt.Errorf("%s: validate response: %w: %w", op, ErrResultUnknown, err)
	}

	return &payment, nil
}

func (c *Client) GetPaymentByID(ctx context.Context, paymentID string) (*Payment, error) {
	const op = "yookassa.client.GetPaymentByID"

	paymentID = strings.TrimSpace(paymentID)
	if paymentID == "" {
		return nil, fmt.Errorf("%s: invalid payment id: %q: %w", op, paymentID, ErrInvalidArgument)
	}

	fullURL, err := url.JoinPath(c.apiURL, "payments", url.PathEscape(paymentID))
	if err != nil {
		return nil, fmt.Errorf("%s: failed to join url path: %w", op, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: create request: %w", op, err)
	}

	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(c.shopID, c.secretKey)

	res, err := c.httpClient.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("%s: send request: %w", op, ctxErr)
		}
		return nil, fmt.Errorf("%s: %w: %w", op, ErrProviderUnavailable, err)
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		cause := mapStatusCode(res.StatusCode)

		if res.StatusCode >= http.StatusInternalServerError && res.StatusCode <= 599 {
			cause = ErrProviderUnavailable
		}

		apiError := APIError{
			StatusCode: res.StatusCode,
			Cause:      cause,
		}

		var errResponse apiErrorResponse
		if err := json.NewDecoder(res.Body).Decode(&errResponse); err == nil {
			apiError.ID = errResponse.ID
			apiError.Code = errResponse.Code
			apiError.Description = errResponse.Description
			apiError.Parameter = errResponse.Parameter
		}

		return nil, fmt.Errorf("%s: %w", op, &apiError)
	}

	var payment Payment
	if err := json.NewDecoder(res.Body).Decode(&payment); err != nil {
		return nil, fmt.Errorf("%s: decode payment response: %w: %w", op, ErrInvalidResponse, err)
	}

	if err := validateGetPaymentResponse(&payment, paymentID); err != nil {
		return nil, fmt.Errorf("%s: validate response: %w", op, err)
	}

	return &payment, nil
}

func (c Client) buildCreatePaymentRequest(params CreatePaymentParams) *createPaymentRequest {
	paymentReq := createPaymentRequest{
		Money: Money{
			Value:    formatAmount(params.AmountCents),
			Currency: strings.TrimSpace(params.Currency),
		},
		Capture:     true,
		Description: params.Description,
		Metadata: Metadata{
			OrderID:   params.OrderID.String(),
			PaymentID: params.LocalPaymentID.String(),
		},
		Confirmation: confirmationRequest{
			Type:      "redirect",
			ReturnURL: c.returnURL,
		},
	}

	return &paymentReq
}

func validatePayment(p *Payment) error {
	if p == nil {
		return fmt.Errorf("nil payment response: %w", ErrInvalidResponse)
	}

	if strings.TrimSpace(p.ID) == "" {
		return fmt.Errorf("empty payment id: %w", ErrInvalidResponse)
	}

	if !isKnownPaymentStatus(p.Status) {
		return fmt.Errorf("unknown payment status: %q: %w", p.Status, ErrInvalidResponse)
	}

	if strings.TrimSpace(p.Money.Currency) == "" {
		return fmt.Errorf("empty payment currency: %s: %w", p.Money.Currency, ErrInvalidResponse)
	}

	if strings.TrimSpace(p.Money.Value) == "" {
		return fmt.Errorf("empty payment money value: %s: %w", p.Money.Value, ErrInvalidResponse)
	}

	if p.CreatedAt.IsZero() {
		return fmt.Errorf("empty payment created_at: %w", ErrInvalidResponse)
	}

	return nil
}

func validateCreatePaymentResponse(p *Payment, req *createPaymentRequest) error {
	if err := validatePayment(p); err != nil {
		return err
	}

	if p.Money.Value != req.Money.Value {
		return fmt.Errorf(
			"amount mismatch: expected %s, got %s: %w",
			req.Money.Value,
			p.Money.Value,
			ErrInvalidResponse,
		)
	}

	if p.Money.Currency != req.Money.Currency {
		return fmt.Errorf(
			"currency mismatch: expected %s, got %s: %w",
			req.Money.Currency,
			p.Money.Currency,
			ErrInvalidResponse,
		)
	}

	if p.Metadata.OrderID != req.Metadata.OrderID {
		return fmt.Errorf(
			"metadata order_id mismatch: %w",
			ErrInvalidResponse,
		)
	}

	if p.Metadata.PaymentID != req.Metadata.PaymentID {
		return fmt.Errorf(
			"metadata payment_id mismatch: %w",
			ErrInvalidResponse,
		)
	}

	if p.Status == StatusPending {
		if p.Confirmation == nil {
			return fmt.Errorf(
				"pending payment has no confirmation: %w",
				ErrInvalidResponse,
			)
		}

		if p.Confirmation.Type != req.Confirmation.Type {
			return fmt.Errorf(
				"confirmation type mismatch: expected %s, got %s: %w",
				req.Confirmation.Type,
				p.Confirmation.Type,
				ErrInvalidResponse,
			)
		}

		if strings.TrimSpace(p.Confirmation.ConfirmationURL) == "" {
			return fmt.Errorf(
				"pending payment has empty confirmation URL: %w",
				ErrInvalidResponse,
			)
		}
	}

	return nil
}

func validateGetPaymentResponse(p *Payment, paymentID string) error {
	if err := validatePayment(p); err != nil {
		return err
	}

	if p.ID != paymentID {
		return fmt.Errorf("mismatch payment id: expected %s, got: %s: %w", paymentID, p.ID, ErrInvalidResponse)
	}

	return nil
}

func isKnownPaymentStatus(status PaymentStatus) bool {
	switch status {
	case StatusPending,
		StatusSucceeded,
		StatusCanceled,
		StatusWaitingForCapture:
		return true
	}
	return false
}

func formatAmount(amount int64) string {
	return fmt.Sprintf("%d.%02d", amount/100, amount%100)
}

func mapStatusCode(statusCode int) error {
	switch {
	case statusCode >= http.StatusInternalServerError && statusCode <= 599:
		return ErrResultUnknown

	case statusCode == http.StatusBadRequest:
		return ErrInvalidRequest
	case statusCode == http.StatusUnauthorized:
		return ErrInvalidCredentials
	case statusCode == http.StatusForbidden:
		return ErrForbidden
	case statusCode == http.StatusNotFound:
		return ErrNotFound
	case statusCode == http.StatusTooManyRequests:
		return ErrRateLimited

	default:
		return ErrUnexpectedResponse
	}
}

func (p *CreatePaymentParams) validate() error {
	const op = "yookassa.CreatePaymentParams.validate"

	if p.AmountCents <= 0 {
		return fmt.Errorf("%s: %w", op, ErrInvalidArgument)
	}

	if p.OrderID == uuid.Nil || p.LocalPaymentID == uuid.Nil || p.IdempotencyKey == uuid.Nil {
		return fmt.Errorf("%s: %w", op, ErrInvalidArgument)
	}

	if strings.TrimSpace(p.Currency) == "" {
		return fmt.Errorf("%s: %w", op, ErrInvalidArgument)
	}

	return nil
}
