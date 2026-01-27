package subscription

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"
)

// Handler handles subscription HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates subscription handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
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
// Subscribe handles POST /subscriptions
func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := validator.Validate(req); err != nil {
		response.ValidationError(w, err)
		return
	}

	userID := middleware.GetUserID(ctx)
	if userID == uuid.Nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

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

	plan, _ := h.service.repo.GetPlanByID(ctx, sub.PlanID)

	response.Created(
		w,
		SubscriptionResponseFromEntity(sub, plan),
	)
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
