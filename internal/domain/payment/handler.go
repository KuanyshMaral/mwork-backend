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

// KaspiWebhookPayload represents incoming webhook data
type KaspiWebhookPayload struct {
	EventType string  `json:"event_type"` // payment.success, payment.failed
	OrderID   string  `json:"order_id"`
	PaymentID string  `json:"payment_id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
	Timestamp string  `json:"timestamp"`
}

// HandleKaspiWebhook processes payment status updates from Kaspi
func (h *Handler) HandleKaspiWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Read raw body for signature verification
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read webhook body")
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "failed to read body")
		return
	}
	defer r.Body.Close()

	// Verify signature
	signature := r.Header.Get("X-Signature")
	if signature == "" {
		log.Warn().Msg("Webhook received without signature")
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "signature required")
		return
	}

	if !kaspi.VerifySignature(body, signature, h.kaspiSecret) {
		log.Warn().
			Str("signature", signature).
			Msg("Invalid webhook signature")
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid signature")
		return
	}

	// Parse JSON payload
	var payload KaspiWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Error().Err(err).Msg("Failed to parse webhook payload")
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}

	// Validate required fields
	if payload.OrderID == "" {
		log.Warn().Msg("Webhook payload missing order_id")
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "order_id is required")
		return
	}
	if payload.Status == "" {
		log.Warn().Msg("Webhook payload missing status")
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "status is required")
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
		// Try to infer from status field
		switch payload.Status {
		case "success", "completed", "paid":
			internalStatus = "completed"
		case "failed", "cancelled", "declined":
			internalStatus = "failed"
		case "pending":
			internalStatus = "pending"
		default:
			log.Warn().
				Str("event_type", payload.EventType).
				Str("status", payload.Status).
				Msg("Unknown webhook status")
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "unknown status")
			return
		}
	}

	log.Info().
		Str("order_id", payload.OrderID).
		Str("payment_id", payload.PaymentID).
		Str("status", internalStatus).
		Float64("amount", payload.Amount).
		Msg("Processing Kaspi webhook")

	// Update payment status via service
	if err := h.service.UpdatePaymentByKaspiOrderID(ctx, payload.OrderID, internalStatus); err != nil {
		if err == ErrPaymentNotFound {
			log.Warn().
				Str("order_id", payload.OrderID).
				Msg("Payment not found for webhook - idempotency, returning success")
			response.JSON(w, http.StatusOK, map[string]string{"status": "processed"})
			return
		}
		log.Error().
			Err(err).
			Str("order_id", payload.OrderID).
			Msg("Failed to update payment status")
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error")
		return
	}

	log.Info().
		Str("order_id", payload.OrderID).
		Str("status", internalStatus).
		Msg("Webhook processed successfully")

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
