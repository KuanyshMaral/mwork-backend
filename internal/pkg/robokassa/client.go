package robokassa

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Config holds RoboKassa configuration
type Config struct {
	MerchantLogin string        // Merchant login (MrchLogin)
	Password1     string        // Password #1 for payment initialization
	Password2     string        // Password #2 for webhook verification (ResultURL)
	TestMode      bool          // Test mode flag
	HashAlgo      HashAlgorithm // Hash algorithm: MD5 or SHA256 (default: SHA256)
	Timeout       time.Duration
}

// Client represents RoboKassa payment gateway client
type Client struct {
	config Config
}

// CreatePaymentRequest represents payment creation request
type CreatePaymentRequest struct {
	Amount         float64           // Payment amount
	InvID          int64             // Invoice ID (unique order identifier)
	Description    string            // Payment description
	Email          string            // Optional: user email
	Culture        string            // Optional: interface language (ru, en)
	ExpirationDate string            // Optional: payment expiration (ISO 8601)
	OutSum         string            // Optional: pre-calculated amount string
	Shp            map[string]string // Optional: custom parameters (shp_*)
}

// CreatePaymentResponse represents payment creation response
// RoboKassa doesn't return JSON - we return the payment URL directly
type CreatePaymentResponse struct {
	PaymentURL string // URL to redirect user for payment
	InvID      int64  // Invoice ID
}

// NewClient creates new RoboKassa client
func NewClient(cfg Config) *Client {
	return &Client{
		config: cfg,
	}
}

// CreatePayment generates payment URL for user redirect
// Unlike Kaspi's API call, RoboKassa uses GET redirect with signature
func (c *Client) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*CreatePaymentResponse, error) {
	// Validation
	if req.Amount <= 0 {
		return nil, fmt.Errorf("validation error: amount must be > 0")
	}
	if req.InvID <= 0 {
		return nil, fmt.Errorf("validation error: invoice ID must be > 0")
	}
	if strings.TrimSpace(c.config.MerchantLogin) == "" {
		return nil, fmt.Errorf("robokassa config error: merchant_login is empty")
	}
	if strings.TrimSpace(c.config.Password1) == "" {
		return nil, fmt.Errorf("robokassa config error: password1 is empty")
	}

	// Format amount with 2 decimal places
	outSum := fmt.Sprintf("%.2f", req.Amount)
	if req.OutSum != "" {
		outSum = req.OutSum
	}

	// Determine hash algorithm (default to SHA256 if not set)
	algo := c.config.HashAlgo
	if algo == "" {
		algo = HashSHA256
	}

	// Build shp map with "Shp_" prefixed keys for signature
	shpForSig := make(map[string]string)
	for k, v := range req.Shp {
		shpKey := k
		if !strings.HasPrefix(strings.ToLower(k), "shp_") {
			shpKey = "Shp_" + k
		}
		shpForSig[shpKey] = v
	}

	// Build signature: Hash(MerchantLogin:OutSum:InvId:Password1[:Shp_params])
	// Uses BuildStartSignatureBase to correctly handle SHP params
	signatureBase := BuildStartSignatureBase(
		c.config.MerchantLogin,
		outSum,
		strconv.FormatInt(req.InvID, 10),
		c.config.Password1,
		nil,
		shpForSig,
	)
	signature, err := Sign(signatureBase, algo)
	if err != nil {
		return nil, fmt.Errorf("robokassa: failed to sign payment request: %w", err)
	}
	_ = sort.Search // ensure sort import is used

	// Build payment URL
	baseURL := "https://auth.robokassa.ru/Merchant/Index.aspx"
	if c.config.TestMode {
		baseURL = "https://auth.robokassa.ru/Merchant/Index.aspx" // Same URL, test mode via IsTest param
	}

	params := url.Values{}
	params.Set("MerchantLogin", c.config.MerchantLogin)
	params.Set("OutSum", outSum)
	params.Set("InvId", strconv.FormatInt(req.InvID, 10))
	params.Set("Description", req.Description)
	params.Set("SignatureValue", signature)

	if c.config.TestMode {
		params.Set("IsTest", "1")
	}

	if req.Email != "" {
		params.Set("Email", req.Email)
	}

	if req.Culture != "" {
		params.Set("Culture", req.Culture)
	} else {
		params.Set("Culture", "ru") // Default to Russian
	}

	if req.ExpirationDate != "" {
		params.Set("ExpirationDate", req.ExpirationDate)
	}

	// Add custom parameters
	for k, v := range req.Shp {
		params.Set("Shp_"+k, v)
	}

	paymentURL := baseURL + "?" + params.Encode()

	return &CreatePaymentResponse{
		PaymentURL: paymentURL,
		InvID:      req.InvID,
	}, nil
}
