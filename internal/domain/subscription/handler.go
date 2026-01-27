package subscription

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/kaspi"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"
)

// Handler handles subscription HTTP requests
type Handler struct {
	service        *Service
	paymentService PaymentService
	kaspiClient    *kaspi.Client
	validator      *validator.Validator
	config         *Config
}

// Config holds handler configuration
type Config struct {
	FrontendURL string
	BackendURL  string
}

// PaymentService interface for payment operations
type PaymentService interface {
	CreatePendingPayment(ctx context.Context, userID, subscriptionID uuid.UUID, amount float64, kaspiOrderID, billingPeriod string) error
}

// NewHandler creates subscription handler
func NewHandler(service *Service, paymentService PaymentService, kaspiClient *kaspi.Client, v *validator.Validator, cfg *Config) *Handler {
	return &Handler{
		service:        service,
		paymentService: paymentService,
		kaspiClient:    kaspiClient,
		validator:      v,
		config:         cfg,
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

	plan, err := h.service.GetLimits(r.Context(), userID)
	if err != nil {
		response.InternalError(w)
		return
	}

	// TODO: Get actual usage from photo and response repos
	limits := &LimitsResponse{
		Plan:           string(plan.ID),
		MaxPhotos:      plan.MaxPhotos,
		PhotosUsed:     0, // TODO: Count from photo repo
		MaxResponses:   plan.MaxResponsesMonth,
		ResponsesUsed:  0, // TODO: Count from response repo
		CanChat:        plan.CanChat,
		CanSeeViewers:  plan.CanSeeViewers,
		PrioritySearch: plan.PrioritySearch,
	}

	response.OK(w, limits)
}

// Subscribe handles POST /subscriptions
func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Step 1: Parse and validate request
	var req SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := h.validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	// Step 2: Extract userID from context
	userID := middleware.GetUserID(ctx)
	if userID == uuid.Nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Step 3: Check for existing subscription
	_, existingPlan, err := h.service.GetCurrentSubscription(ctx, userID)
	if err == nil && existingPlan != nil && existingPlan.ID != PlanFree {
		response.Error(w, http.StatusConflict, "already subscribed")
		return
	}

	// Step 4: Get plan details
	var planID PlanID
	switch req.PlanID {
	case "pro":
		planID = PlanPro
	case "agency":
		planID = PlanAgency
	default:
		response.Error(w, http.StatusNotFound, "plan not found")
		return
	}

	plan, err := h.service.GetPlan(ctx, planID)
	if err != nil || plan == nil {
		response.Error(w, http.StatusNotFound, "plan not found")
		return
	}

	// Step 5: Calculate amount based on billing period
	amount := plan.PriceMonthly
	if req.BillingPeriod == "yearly" {
		amount = plan.PriceMonthly * 12 * 0.85 // 15% discount
	}

	// Step 6: Create pending subscription
	sub, err := h.service.CreatePendingSubscription(ctx, userID, planID, req.BillingPeriod)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create pending subscription")
		response.Error(w, http.StatusInternalServerError, "payment initiation failed")
		return
	}

	// Step 7: Generate unique Kaspi order ID
	orderID := uuid.New().String()

	// Step 8: Create pending payment
	err = h.paymentService.CreatePendingPayment(ctx, userID, sub.ID, amount, orderID, req.BillingPeriod)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create pending payment")
		response.Error(w, http.StatusInternalServerError, "payment initiation failed")
		return
	}

	// Step 9: Create Kaspi payment
	kaspiReq := kaspi.CreatePaymentRequest{
		Amount:      amount,
		OrderID:     orderID,
		Description: fmt.Sprintf("MWork %s subscription", req.PlanID),
		ReturnURL:   h.config.FrontendURL + "/payment/success",
		CallbackURL: h.config.BackendURL + "/webhooks/kaspi",
	}

	kaspiResp, err := h.kaspiClient.CreatePayment(ctx, kaspiReq)
	if err != nil {
		log.Error().Err(err).Msg("Kaspi API call failed")
		response.Error(w, http.StatusBadGateway, "payment gateway error")
		return
	}

	// Step 10: Return response with payment URL
	expiresAt := time.Now().Add(30 * time.Minute).Format(time.RFC3339)

	resp := &SubscribeResponse{
		PaymentID:  sub.ID.String(),
		PaymentURL: kaspiResp.PaymentURL,
		Amount:     amount,
		ExpiresAt:  expiresAt,
	}

	log.Info().
		Str("user_id", userID.String()).
		Str("plan_id", req.PlanID).
		Str("order_id", orderID).
		Float64("amount", amount).
		Msg("Subscription payment initiated")

	response.JSON(w, http.StatusCreated, resp)
}
		},
	}

	response.Created(w, paymentInfo)
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
		r.Post("/cancel", h.Cancel)
	})

	return r
}
