package payment

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/mwork/mwork-api/internal/middleware"
	paymentpkg "github.com/mwork/mwork-api/internal/pkg/payment"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/robokassa"
)

// Handler handles payment HTTP requests
type Handler struct {
	service         *Service
	providerFactory *paymentpkg.ProviderFactory
}

// NewHandler creates payment handler with provider factory
func NewHandler(service *Service, factory *paymentpkg.ProviderFactory) *Handler {
	return &Handler{
		service:         service,
		providerFactory: factory,
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

// HandleRoboKassaWebhook processes payment webhooks from RoboKassa (ResultURL)
// CRITICAL: This is called AFTER successful payment on RoboKassa gateway
func (h *Handler) HandleRoboKassaWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse form data (RoboKassa sends form-encoded data, not JSON)
	if err := r.ParseForm(); err != nil {
		log.Error().Err(err).Msg("Failed to parse form data")
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "failed to parse form")
		return
	}

	// Parse webhook payload
	payload, err := robokassa.ParseWebhookForm(r.Form)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse robokassa webhook")
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	// Get provider from factory
	provider, err := h.providerFactory.Get("robokassa")
	if err != nil {
		log.Error().Err(err).Msg("RoboKassa provider not registered")
		response.Error(w, http.StatusInternalServerError, "PROVIDER_ERROR", "robokassa not configured")
		return
	}

	// Verify webhook signature using provider
	signature := r.FormValue("SignatureValue")
	if !provider.VerifyWebhook(payload, signature) {
		log.Warn().
			Str("invoice_id", fmt.Sprintf("%d", payload.InvId)).
			Str("signature", signature).
			Msg("Invalid webhook signature")
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid signature")
		return
	}

	log.Info().
		Int64("invoice_id", payload.InvId).
		Str("amount", payload.OutSum).
		Msg("Processing RoboKassa webhook")

	// Update payment status via service (idempotency handled inside)
	if err := h.service.UpdatePaymentByInvoiceID(ctx, payload.InvId, "completed"); err != nil {
		log.Error().
			Err(err).
			Int64("invoice_id", payload.InvId).
			Msg("Failed to update payment")
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to process webhook")
		return
	}

	log.Info().
		Int64("invoice_id", payload.InvId).
		Msg("RoboKassa webhook processed successfully")

	// RoboKassa expects HTTP 200 OK
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// WebhookHandler is generic webhook handler for any provider
// Routes webhook to appropriate handler based on provider name
func (h *Handler) WebhookHandler(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")

	log.Info().Str("provider", providerName).Msg("Webhook received")

	// Route to specific provider handler
	switch providerName {
	case "robokassa":
		h.HandleRoboKassaWebhook(w, r)
	default:
		log.Warn().Str("provider", providerName).Msg("Unknown payment provider")
		response.Error(w, http.StatusNotFound, "PROVIDER_NOT_FOUND", "unknown provider")
	}
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

// WebhookRoutes returns webhook router (no auth)
func (h *Handler) WebhookRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/{provider}", h.WebhookHandler)
	return r
}
