package payment

import (
	"context"
	"fmt"
)

// Provider constants
const (
	ProviderRoboKassa = "robokassa"
	ProviderKaspi     = "kaspi"
)

// PaymentProvider defines the interface that all payment providers must implement
// This abstraction allows switching between Kaspi, RoboKassa, and future providers
type PaymentProvider interface {
	// CreatePayment initiates a payment and returns payment URL for user redirect
	CreatePayment(ctx context.Context, req ProviderPaymentRequest) (*ProviderPaymentResponse, error)

	// VerifyWebhook validates webhook signature and returns whether it's authentic
	VerifyWebhook(rawData interface{}, signature string) bool

	// ParseWebhook parses provider-specific webhook data into standardized format
	ParseWebhook(rawData interface{}) (*WebhookEvent, error)

	// Name returns the provider identifier (e.g., "robokassa", "kaspi")
	Name() string
}

// ProviderPaymentRequest is a standardized payment creation request
type ProviderPaymentRequest struct {
	Amount      float64           // Payment amount in currency
	OrderID     string            // Internal order/subscription ID
	InvoiceID   int64             // Unique invoice number (for RoboKassa)
	Description string            // Payment description
	UserEmail   string            // Optional: user email
	ReturnURL   string            // URL to redirect after payment
	CallbackURL string            // Webhook URL for server notification
	Metadata    map[string]string // Provider-specific custom parameters
}

// ProviderPaymentResponse is a standardized payment creation response
type ProviderPaymentResponse struct {
	PaymentID  string // Provider's payment identifier (may be empty for redirect-based)
	PaymentURL string // URL to redirect user for payment
	InvoiceID  int64  // Invoice ID (for RoboKassa)
	Status     string // Initial payment status
}

// WebhookEvent is a standardized webhook event across all providers
type WebhookEvent struct {
	Provider   string  // Provider name ("robokassa", "kaspi")
	EventType  string  // Event type (e.g., "payment.success")
	OrderID    string  // Internal order ID
	InvoiceID  int64   // Invoice ID (RoboKassa) or 0
	ExternalID string  // Provider's transaction/payment ID
	Amount     float64 // Payment amount
	Status     string  // Payment status (standardized: "completed", "failed", "pending")
	RawData    string  // Original webhook payload for debugging
}

// ProviderFactory creates payment provider instances
type ProviderFactory struct {
	providers map[string]PaymentProvider
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{
		providers: make(map[string]PaymentProvider),
	}
}

// Register adds a payment provider to the factory
func (f *ProviderFactory) Register(name string, provider PaymentProvider) {
	f.providers[name] = provider
}

// Get retrieves a payment provider by name
func (f *ProviderFactory) Get(name string) (PaymentProvider, error) {
	provider, exists := f.providers[name]
	if !exists {
		return nil, fmt.Errorf("payment provider '%s' not found", name)
	}
	return provider, nil
}

// List returns all registered provider names
func (f *ProviderFactory) List() []string {
	names := make([]string, 0, len(f.providers))
	for name := range f.providers {
		names = append(names, name)
	}
	return names
}

// MapStatusToInternal converts provider-specific status to internal status
// Returns one of: "pending", "completed", "failed", "refunded"
func MapStatusToInternal(providerStatus string, provider string) string {
	// Standardize to lowercase for comparison
	status := ""
	for _, r := range providerStatus {
		if r >= 'A' && r <= 'Z' {
			status += string(r + 32)
		} else {
			status += string(r)
		}
	}

	// Common success statuses
	switch status {
	case "success", "completed", "paid", "approved", "authorized":
		return "completed"
	case "failed", "cancelled", "declined", "rejected", "error":
		return "failed"
	case "pending", "processing", "awaiting":
		return "pending"
	case "refunded", "reversed":
		return "refunded"
	default:
		// Unknown status defaults to pending
		return "pending"
	}
}
