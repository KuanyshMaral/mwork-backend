package payment

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/kaspi"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

// Handler handles payment HTTP requests
type Handler struct {
	service     *Service
	kaspiSecret string
}

// NewHandler creates payment handler
func NewHandler(service *Service, kaspiSecret string) *Handler {
	return &Handler{
		service:     service,
		kaspiSecret: kaspiSecret,
	}
}

// KaspiWebhookPayload represents incoming webhook data
type KaspiWebhookPayload struct {
	EventType string  `json:"event_type"` // payment.success, payment.failed
	OrderID   string  `json:"order_id"`
	PaymentID string  `json:"payment_id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
	Timestamp string  `json:"timestamp"`
}

// GetHistory handles GET /payments
func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	payments, err := h.service.GetPaymentHistory(r.Context(), userID, limit, offset)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, payments)
}

// HandleKaspiWebhook processes payment status updates from Kaspi
func (h *Handler) HandleKaspiWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Read body once
	body, err := io.ReadAll(r.Body)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	// Verify signature
	signature := r.Header.Get("X-Signature")
	if signature == "" {
		log.Warn().Msg("Missing X-Signature header")
		response.Error(w, http.StatusUnauthorized, "signature required")
		return
	}

	if !kaspi.VerifySignature(body, signature, h.kaspiSecret) {
		log.Warn().Str("signature", signature).Msg("Invalid signature")
		response.Error(w, http.StatusUnauthorized, "invalid signature")
		return
	}

	// Parse JSON from already-read body
	var payload KaspiWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Error().Err(err).Msg("Invalid JSON payload")
		response.Error(w, http.StatusBadRequest, "invalid payload")
		return
	}

	// Validate required fields
	if payload.OrderID == "" || payload.PaymentID == "" || payload.Status == "" {
		log.Warn().Interface("payload", payload).Msg("Missing required fields")
		response.Error(w, http.StatusBadRequest, "invalid payload")
		return
	}

	// Map Kaspi status to internal status
	var internalStatus string
	switch payload.EventType {
	case "payment.success":
		internalStatus = "completed"
	case "payment.failed":
		internalStatus = "failed"
	case "payment.pending":
		internalStatus = "pending"
	default:
		log.Warn().Str("event_type", payload.EventType).Msg("Unknown event type")
		internalStatus = "pending"
	}

	// Update payment status
	err = h.service.UpdatePaymentByKaspiOrderID(ctx, payload.OrderID, internalStatus)
	if err != nil {
		// Payment not found - log warning for idempotency
		log.Warn().
			Str("order_id", payload.OrderID).
			Str("payment_id", payload.PaymentID).
			Msg("Payment not found (idempotency check)")
		response.Error(w, http.StatusNotFound, "payment not found")
		return
	}

	// Return success response
	log.Info().
		Str("order_id", payload.OrderID).
		Str("status", internalStatus).
		Msg("Payment status updated")

	response.JSON(w, http.StatusOK, map[string]string{"status": "processed"})
}

// Webhook handles POST /webhooks/payment
// This is called by payment providers (Kaspi, etc.)
func (h *Handler) Webhook(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")

	// Parse webhook data (structure varies by provider)
	var data struct {
		TransactionID string  `json:"transaction_id"`
		ExternalID    string  `json:"external_id"`
		Status        string  `json:"status"`
		Amount        float64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		response.BadRequest(w, "Invalid webhook data")
		return
	}

	// TODO: Verify webhook signature based on provider
	// For Kaspi: verify HMAC signature
	// For now, just process the webhook

	externalID := data.ExternalID
	if externalID == "" {
		externalID = data.TransactionID
	}

	if err := h.service.HandleWebhook(r.Context(), provider, externalID, data.Status); err != nil {
		// Don't expose internal errors to webhook caller
		response.OK(w, map[string]string{"status": "error", "message": "payment not found"})
		return
	}

	response.OK(w, map[string]string{"status": "ok"})
}

// Routes returns payment router
func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// Payment history (protected)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Get("/", h.GetHistory)
	})

	return r
}

// WebhookRoutes returns webhook router (no auth, but signature verification)
func (h *Handler) WebhookRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/{provider}", h.Webhook)
	return r
}
