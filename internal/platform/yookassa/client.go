package yookassa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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

	normalizedCfg := YookassaClientConfig{
		APIURL:     apiURL,
		ShopID:     shopID,
		SecretKey:  secretKey,
		ReturnURL:  returnURL,
		HTTPClient: cfg.HTTPClient,
	}

	if err := normalizedCfg.validate(); err != nil {
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

func (c YookassaClientConfig) validate() error {
	if c.HTTPClient == nil {
		return fmt.Errorf("yookassa client: nil HTTP client: %w", ErrInvalidArgument)
	}

	parsedAPIURL, err := url.ParseRequestURI(c.APIURL)
	if err != nil {
		return fmt.Errorf("yookassa client: invalid api URL: %w", ErrInvalidArgument)
	}
	if parsedAPIURL.Scheme != "http" && parsedAPIURL.Scheme != "https" {
		return fmt.Errorf("yookassa client: unsupported api URL scheme: %w", ErrInvalidArgument)
	}
	if parsedAPIURL.Host == "" {
		return fmt.Errorf("yookassa client: empty api URL host: %w", ErrInvalidArgument)
	}

	if c.ShopID == "" {
		return fmt.Errorf("yookassa client: empty shop id: %w", ErrInvalidArgument)
	}

	if c.SecretKey == "" {
		return fmt.Errorf("yookassa client: empty secret key: %w", ErrInvalidArgument)
	}

	parsedReturnURL, err := url.ParseRequestURI(c.ReturnURL)
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

func (c *Client) doPaymentRequest(
	ctx context.Context,
	method string,
	path string,
	idempotenceKey string,
	body any,
	op string,
) (*Payment, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("%s: marshal body: %w", op, err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	fullURL, err := url.JoinPath(c.apiURL, path)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to join url path: %w", op, err)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("%s: create request: %w", op, err)
	}

	req.SetBasicAuth(c.shopID, c.secretKey)
	req.Header.Set("Content-Type", "application/json")

	if idempotenceKey != "" {
		req.Header.Set("Idempotence-Key", idempotenceKey)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("%s: send request: %w", op, ctxErr)
		}
		return nil, fmt.Errorf("%s: %w: %w", op, ErrProviderUnavailable, err)
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		cause := mapHTTPStatusError(res.StatusCode)

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

	if err := validatePayment(&payment); err != nil {
		return nil, fmt.Errorf("%s: validate payment response: %w", op, err)
	}

	return &payment, nil
}
