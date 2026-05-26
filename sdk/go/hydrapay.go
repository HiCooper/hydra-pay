package hydrapay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.hydrapay.com/v1"

// Client is the HydraPay API client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string

	Payments      *PaymentService
	Checkout      *CheckoutService
	Refunds       *RefundService
	Subscriptions *SubscriptionService
	Webhooks      *WebhookService
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL sets a custom base URL for the API.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(url, "/") }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// NewClient creates a new HydraPay API client with the given API key.
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    defaultBaseURL,
		apiKey:     apiKey,
	}
	for _, o := range opts {
		o(c)
	}
	c.Payments = &PaymentService{client: c}
	c.Checkout = &CheckoutService{client: c}
	c.Refunds = &RefundService{client: c}
	c.Subscriptions = &SubscriptionService{client: c}
	c.Webhooks = &WebhookService{}
	return c
}

func (c *Client) do(method, path string, body interface{}, idempotency *IdempotencyParams, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("hydrapay: failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("hydrapay: failed to create request: %w", err)
	}

	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if idempotency != nil && idempotency.Key != "" {
		req.Header.Set("Idempotency-Key", idempotency.Key)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("hydrapay: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("hydrapay: failed to read response: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return fmt.Errorf("hydrapay: failed to parse response: %w", err)
	}

	if !apiResp.Success {
		err := &APIError{StatusCode: resp.StatusCode}
		if apiResp.Error != nil {
			err.Code = apiResp.Error.Code
			err.Message = apiResp.Error.Message
		}
		return err
	}

	if result != nil {
		dataBytes, err := json.Marshal(apiResp.Data)
		if err != nil {
			return fmt.Errorf("hydrapay: failed to re-marshal data: %w", err)
		}
		if err := json.Unmarshal(dataBytes, result); err != nil {
			return fmt.Errorf("hydrapay: failed to unmarshal data: %w", err)
		}
	}

	return nil
}
