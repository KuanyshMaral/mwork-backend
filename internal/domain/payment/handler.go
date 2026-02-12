package payment

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

// Handler handles payment HTTP requests
type Handler struct {
	service *Service
}

type InitRobokassaRequest struct {
	SubscriptionID string `json:"subscription_id"`
	Amount         string `json:"amount"`
	Description    string `json:"description"`
}

func (h *Handler) InitRobokassaPayment(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req InitRobokassaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	subscriptionID, err := parseUUID(req.SubscriptionID)
	if err != nil {
		response.BadRequest(w, "invalid subscription_id")
		return
	}
	if _, err := strconv.ParseFloat(req.Amount, 64); err != nil {
		response.BadRequest(w, "invalid amount")
		return
	}
	out, err := h.service.InitRobokassaPayment(r.Context(), InitRobokassaPaymentRequest{
		UserID:         userID,
		SubscriptionID: subscriptionID,
		Amount:         req.Amount,
		Description:    req.Description,
	})
	if err != nil {
		log.Error().Err(err).Msg("robokassa init failed")
		response.InternalError(w)
		return
	}
	response.OK(w, out)
}

func (h *Handler) RobokassaResult(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid form data")
		return
	}
	outSum := r.Form.Get("OutSum")
	invID := r.Form.Get("InvId")
	signature := r.Form.Get("SignatureValue")
	if outSum == "" || invID == "" || signature == "" {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "required fields are missing")
		return
	}

	shp := map[string]string{}
	raw := map[string]string{}
	for key, values := range r.Form {
		if len(values) == 0 {
			continue
		}
		raw[key] = values[0]
		if strings.HasPrefix(strings.ToLower(key), "shp_") {
			shp[key] = values[0]
		}
	}

	err := h.service.ProcessRobokassaResult(r.Context(), outSum, invID, signature, shp, raw)
	if err != nil {
		log.Warn().Err(err).Str("inv_id", invID).Msg("robokassa result rejected")
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid callback")
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte("OK" + invID))
}

func (h *Handler) RobokassaSuccess(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]string{"status": "processing", "message": "Оплата принята, проверяем статус"})
}

func (h *Handler) RobokassaFail(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]string{"status": "failed", "message": "Платеж отменен или не завершен"})
}

// NewHandler creates payment handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
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

// Webhook handles POST /webhooks/payment
// This is called by payment providers.
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
		r.Post("/robokassa/init", h.InitRobokassaPayment)
		r.Get("/robokassa/success", h.RobokassaSuccess)
		r.Post("/robokassa/success", h.RobokassaSuccess)
		r.Get("/robokassa/fail", h.RobokassaFail)
		r.Post("/robokassa/fail", h.RobokassaFail)
	})

	return r
}

// WebhookRoutes returns webhook router (no auth, but signature verification)
func (h *Handler) WebhookRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/{provider}", h.Webhook)
	r.Post("/robokassa/result", h.RobokassaResult)
	r.Get("/robokassa/result", h.RobokassaResult)
	return r
}

func parseUUID(raw string) (uuid.UUID, error) {
	return uuid.Parse(raw)
}
