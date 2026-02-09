package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/mwork/mwork-api/internal/pkg/robokassa"
)

// RoboKassaProvider implements PaymentProvider interface for RoboKassa
type RoboKassaProvider struct {
	client *robokassa.Client
	config robokassa.Config
}

// NewRoboKassaProvider creates a new RoboKassa payment provider
func NewRoboKassaProvider(config robokassa.Config) *RoboKassaProvider {
	return &RoboKassaProvider{
		client: robokassa.NewClient(config),
		config: config,
	}
}

// CreatePayment initiates payment and returns payment URL
func (p *RoboKassaProvider) CreatePayment(ctx context.Context, req ProviderPaymentRequest) (*ProviderPaymentResponse, error) {
	// RoboKassa requires numeric invoice ID
	invoiceID := req.InvoiceID
	if invoiceID == 0 {
		return nil, fmt.Errorf("robokassa requires numeric invoice_id")
	}

	// Build RoboKassa-specific request
	roboReq := robokassa.CreatePaymentRequest{
		Amount:      req.Amount,
		InvID:       invoiceID,
		Description: req.Description,
		Email:       req.UserEmail,
		Culture:     "ru", // Default to Russian
		Shp:         make(map[string]string),
	}

	// Add order_id as custom parameter for tracking
	if req.OrderID != "" {
		roboReq.Shp["order_id"] = req.OrderID
	}

	// Add any additional metadata
	for k, v := range req.Metadata {
		roboReq.Shp[k] = v
	}

	// Create payment (generates URL)
	resp, err := p.client.CreatePayment(ctx, roboReq)
	if err != nil {
		return nil, fmt.Errorf("robokassa create payment failed: %w", err)
	}

	return &ProviderPaymentResponse{
		PaymentID:  strconv.FormatInt(resp.InvID, 10), // Use invoice ID as payment ID
		PaymentURL: resp.PaymentURL,
		InvoiceID:  resp.InvID,
		Status:     "pending",
	}, nil
}

// VerifyWebhook validates webhook signature
func (p *RoboKassaProvider) VerifyWebhook(rawData interface{}, signature string) bool {
	// Parse webhook payload
	payload, ok := rawData.(*robokassa.WebhookPayload)
	if !ok {
		return false
	}

	// Verify using Password2 (ResultURL)
	return robokassa.VerifyResultSignature(
		payload.OutSum,
		payload.InvId,
		signature,
		p.config.Password2,
		payload.Shp,
	)
}

// ParseWebhook converts RoboKassa webhook to standardized format
func (p *RoboKassaProvider) ParseWebhook(rawData interface{}) (*WebhookEvent, error) {
	payload, ok := rawData.(*robokassa.WebhookPayload)
	if !ok {
		return nil, fmt.Errorf("invalid webhook data type for robokassa")
	}

	// Parse amount
	amount, err := strconv.ParseFloat(payload.OutSum, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount in webhook: %w", err)
	}

	// Extract order_id from custom parameters
	orderID := payload.Shp["order_id"]

	// Serialize raw data for debugging
	rawJSON, _ := json.Marshal(payload)

	return &WebhookEvent{
		Provider:   "robokassa",
		EventType:  "payment.success", // RoboKassa only sends success webhooks
		OrderID:    orderID,
		InvoiceID:  payload.InvId,
		ExternalID: strconv.FormatInt(payload.InvId, 10),
		Amount:     amount,
		Status:     "completed", // ResultURL only called on success
		RawData:    string(rawJSON),
	}, nil
}

// Name returns provider identifier
func (p *RoboKassaProvider) Name() string {
	return "robokassa"
}
