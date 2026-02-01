package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"
)

// Handler handles subscription HTTP requests
type Handler struct {
	service        *Service
	paymentService PaymentService
	kaspiClient    KaspiClient
	config         *Config
}

// PaymentService defines payment operations needed by subscription
type PaymentService interface {
	CreatePayment(ctx context.Context, userID, subscriptionID uuid.UUID, amount float64, provider string) (*Payment, error)
}

// KaspiClient defines Kaspi API operations
type KaspiClient interface {
	CreatePayment(ctx context.Context, req KaspiPaymentRequest) (*KaspiPaymentResponse, error)
}

// Payment represents payment entity
type Payment struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	SubscriptionID uuid.NullUUID
	Amount         float64
	KaspiOrderID   string
	Status         string
	CreatedAt      time.Time
}

// KaspiPaymentRequest for creating Kaspi payment
type KaspiPaymentRequest struct {
	Amount      float64
	OrderID     string
	Description string
	ReturnURL   string
	CallbackURL string
}

// KaspiPaymentResponse from Kaspi API
type KaspiPaymentResponse struct {
	PaymentID  string
	PaymentURL string
	Status     string
}

// Config holds application configuration
type Config struct {
	FrontendURL string
	BackendURL  string
}

// NewHandler creates subscription handler
func NewHandler(service *Service, paymentService PaymentService, kaspiClient KaspiClient, config *Config) *Handler {
	return &Handler{
		service:        service,
		paymentService: paymentService,
		kaspiClient:    kaspiClient,
		config:         config,
	}
}

// ListPlans handles GET /subscriptions/plans
func (h *Handler) ListPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.service.GetPlans(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}

	items := make([]*PlanResponse, len(plans))
	for i, p := range plans {
		items[i] = PlanResponseFromEntity(p)
	}

	response.OK(w, items)
}

// GetCurrent handles GET /subscriptions/current
func (h *Handler) GetCurrent(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	sub, plan, err := h.service.GetCurrentSubscription(r.Context(), userID)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, SubscriptionResponseFromEntity(sub, plan))
}

// GetLimits handles GET /subscriptions/limits
func (h *Handler) GetLimits(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	// Use the new comprehensive GetLimitsWithUsage method
	limitsData, err := h.service.GetLimitsWithUsage(r.Context(), userID)
	if err != nil {
		response.InternalError(w)
		return
	}

	// Get plan info for additional fields
	plan, err := h.service.GetPlanLimits(r.Context(), userID)
	if err != nil {
		response.InternalError(w)
		return
	}

	// Build response with both usage and plan features
	limits := &LimitsResponse{
		PlanID:         string(plan.ID),
		MaxPhotos:      limitsData.PhotosLimit,
		PhotosUsed:     limitsData.PhotosUsed,
		MaxResponses:   limitsData.ResponsesLimit,
		ResponsesUsed:  limitsData.ResponsesUsed,
		CanChat:        plan.CanChat,
		CanSeeViewers:  plan.CanSeeViewers,
		PrioritySearch: plan.PrioritySearch,
	}

	response.OK(w, limits)
}

// Subscribe handles POST /subscriptions/subscribe
func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse and validate request
	var req SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	// Extract userID from context
	userID := middleware.GetUserID(ctx)
	if userID == uuid.Nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	// Get plan details first to calculate amount
	planID := PlanID(req.PlanID)
	plan, err := h.service.GetPlan(ctx, planID)
	if err != nil {
		if err == ErrPlanNotFound {
			response.NotFound(w, "plan not found")
			return
		}
		response.InternalError(w)
		return
	}

	// Calculate amount based on billing period
	var amount float64
	switch req.BillingPeriod {
	case "monthly":
		amount = plan.PriceMonthly
	case "yearly":
		// 15% discount for yearly billing
		amount = plan.PriceMonthly * 12 * 0.85
	default:
		response.BadRequest(w, "invalid billing period")
		return
	}

	// Create subscription (status = pending)
	// The service will check for existing subscriptions
	sub, err := h.service.Subscribe(ctx, userID, &req)
	if err != nil {
		switch err {
		case ErrPlanNotFound:
			response.NotFound(w, "plan not found")
		case ErrAlreadySubscribed:
			response.Conflict(w, "already subscribed")
		case ErrInvalidBillingPeriod:
			response.BadRequest(w, "invalid billing period")
		default:
			response.InternalError(w)
		}
		return
	}

	// Generate unique Kaspi order ID
	orderID := uuid.New().String()

	// Create pending payment record
	payment, err := h.paymentService.CreatePayment(ctx, userID, sub.ID, amount, "kaspi")
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "PAYMENT_ERROR", "payment initiation failed")
		return
	}

	// Create Kaspi payment
	kaspiReq := KaspiPaymentRequest{
		Amount:      amount,
		OrderID:     orderID,
		Description: fmt.Sprintf("MWork %s subscription", req.PlanID),
		ReturnURL:   h.config.FrontendURL + "/payment/success",
		CallbackURL: h.config.BackendURL + "/webhooks/kaspi",
	}

	kaspiResp, err := h.kaspiClient.CreatePayment(ctx, kaspiReq)
	if err != nil {
		response.Error(w, http.StatusBadGateway, "GATEWAY_ERROR", "payment gateway error")
		return
	}

	// Calculate expiry time (30 minutes from now)
	expiresAt := time.Now().Add(30 * time.Minute).Format(time.RFC3339)

	// Prepare response
	subscribeResp := struct {
		PaymentID  string  `json:"payment_id"`
		PaymentURL string  `json:"payment_url"`
		Amount     float64 `json:"amount"`
		ExpiresAt  string  `json:"expires_at"`
	}{
		PaymentID:  payment.ID.String(),
		PaymentURL: kaspiResp.PaymentURL,
		Amount:     amount,
		ExpiresAt:  expiresAt,
	}

	response.Created(w, subscribeResp)
}

// Cancel handles POST /subscriptions/cancel
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	var req CancelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = CancelRequest{} // Allow empty body
	}

	userID := middleware.GetUserID(r.Context())
	if err := h.service.Cancel(r.Context(), userID, req.Reason); err != nil {
		switch err {
		case ErrSubscriptionNotFound:
			response.NotFound(w, "No active subscription")
		case ErrCannotCancelFree:
			response.BadRequest(w, "Cannot cancel free subscription")
		default:
			response.InternalError(w)
		}
		return
	}

	response.OK(w, map[string]string{"status": "cancelled"})
}

// Routes returns subscription router
func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// Public routes
	r.Get("/plans", h.ListPlans)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Get("/current", h.GetCurrent)
		r.Get("/limits", h.GetLimits)
		r.Post("/", h.Subscribe)
		r.Post("/subscribe", h.Subscribe)
		r.Post("/cancel", h.Cancel)
	})

	return r
}
