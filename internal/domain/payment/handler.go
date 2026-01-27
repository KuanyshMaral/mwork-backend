package payment

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	limit, offset := 20, 0
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

func (h *Handler) Webhook(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")

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
		response.OK(w, map[string]string{"status": "error"})
		return
	}

	response.OK(w, map[string]string{"status": "ok"})
}

func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Get("/", h.GetHistory)
	})
	return r
}

func (h *Handler) WebhookRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/{provider}", h.Webhook)
	return r
}
