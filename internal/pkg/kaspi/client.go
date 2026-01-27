package kaspi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Config holds Kaspi API configuration
type Config struct {
	BaseURL    string
	MerchantID string
	SecretKey  string
	Timeout    time.Duration
}

// Client represents Kaspi payment gateway client
type Client struct {
	httpClient *http.Client
	config     Config
}

// CreatePaymentRequest represents payment creation request
type CreatePaymentRequest struct {
	Amount      float64 `json:"amount"`
	OrderID     string  `json:"order_id"`
	Description string  `json:"description"`
	ReturnURL   string  `json:"return_url"`
	CallbackURL string  `json:"callback_url"`
}

// CreatePaymentResponse represents payment creation response
type CreatePaymentResponse struct {
	PaymentID  string `json:"payment_id"`
	PaymentURL string `json:"payment_url"`
	Status     string `json:"status"`
}

// NewClient creates new Kaspi API client
func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		config:     cfg,
	}
}

// CreatePayment initiates payment and returns payment URL
func (c *Client) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*CreatePaymentResponse, error) {
	if req.Amount <= 0 {
		return nil, fmt.Errorf("validation error: amount must be > 0")
	}
	if strings.TrimSpace(req.OrderID) == "" {
		return nil, fmt.Errorf("validation error: order_id must be non-empty")
	}
	if c == nil || c.httpClient == nil {
		return nil, fmt.Errorf("kaspi client is not initialized")
	}
	if strings.TrimSpace(c.config.BaseURL) == "" {
		return nil, fmt.Errorf("kaspi config error: base_url is empty")
	}
	if strings.TrimSpace(c.config.MerchantID) == "" {
		return nil, fmt.Errorf("kaspi config error: merchant_id is empty")
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode kaspi request: %w", err)
	}

	base := strings.TrimRight(c.config.BaseURL, "/")
	url := base + "/api/v1/payments/create"

	timeout := c.config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("kaspi api call failed: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.MerchantID)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("kaspi api call failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("kaspi api call failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("kaspi api returned non-2xx status: %d, body: %s", resp.StatusCode, string(body))
	}

	var out CreatePaymentResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("failed to parse kaspi response: %w", err)
	}

	return &out, nil
}
