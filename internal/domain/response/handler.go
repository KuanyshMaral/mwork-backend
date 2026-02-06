package response

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"
)

// Handler handles response HTTP requests
type Handler struct {
	service      *Service
	limitChecker LimitChecker
}

// LimitChecker enforces response limits.
type LimitChecker interface {
	CanApplyToResponse(ctx context.Context, userID uuid.UUID, monthlyApplications int) error
}

// NewHandler creates response handler
func NewHandler(service *Service, limitChecker LimitChecker) *Handler {
	return &Handler{service: service, limitChecker: limitChecker}
}

// Apply handles POST /castings/{id}/responses
// B1: Returns HTTP 402 when user has insufficient credits
func (h *Handler) Apply(w http.ResponseWriter, r *http.Request) {
	castingID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid casting ID")
		return
	}

	var req ApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body
		req = ApplyRequest{}
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	userID := middleware.GetUserID(r.Context())
	if h.limitChecker != nil {
		count, err := h.service.CountMonthlyByUserID(r.Context(), userID)
		if err != nil {
			response.InternalError(w)
			return
		}

		if err := h.limitChecker.CanApplyToResponse(r.Context(), userID, count); err != nil {
			if middleware.WriteLimitExceeded(w, err) {
				return
			}
			response.InternalError(w)
			return
		}
	}

	resp, err := h.service.Apply(r.Context(), userID, castingID, &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrProfileRequired):
			response.BadRequest(w, "You need to create a profile first")
		case errors.Is(err, ErrOnlyModelsCanApply):
			response.Forbidden(w, "Only models can apply to castings")
		case errors.Is(err, ErrCastingNotFound):
			response.NotFound(w, "Casting not found")
		case errors.Is(err, ErrCastingNotActive):
			response.BadRequest(w, "Casting is not active")
		case errors.Is(err, ErrAlreadyApplied):
			response.Conflict(w, "You have already applied to this casting")
		case errors.Is(err, ErrInsufficientCredits):
			// B1: HTTP 402 Payment Required for insufficient credits
			// ✅ FIXED: Added error code parameter
			response.Error(w, http.StatusPaymentRequired, "INSUFFICIENT_CREDITS", "Insufficient credits to apply to this casting")
		default:
			response.InternalError(w)
		}
		return
	}

	response.Created(w, ResponseResponseFromEntity(resp))
}

// ListByCasting handles GET /castings/{id}/responses
func (h *Handler) ListByCasting(w http.ResponseWriter, r *http.Request) {
	castingID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid casting ID")
		return
	}

	// Pagination
	page := 1
	limit := 20
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	pagination := &Pagination{Page: page, Limit: limit}

	userID := middleware.GetUserID(r.Context())
	responses, total, err := h.service.ListByCasting(r.Context(), userID, castingID, pagination)
	if err != nil {
		switch {
		case errors.Is(err, ErrCastingNotFound):
			response.NotFound(w, "Casting not found")
		case errors.Is(err, ErrNotCastingOwner):
			response.Forbidden(w, "Only the casting owner can view responses")
		default:
			response.InternalError(w)
		}
		return
	}

	items := make([]*ResponseResponse, len(responses))
	for i, r := range responses {
		items[i] = ResponseResponseFromEntity(r)
	}

	response.WithMeta(w, items, response.Meta{
		Total:   total,
		Page:    page,
		Limit:   limit,
		Pages:   (total + limit - 1) / limit,
		HasNext: page*limit < total,
		HasPrev: page > 1,
	})
}

// UpdateStatus handles PATCH /responses/{id}/status
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	responseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid response ID")
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	userID := middleware.GetUserID(r.Context())
	resp, err := h.service.UpdateStatus(r.Context(), userID, responseID, Status(req.Status))
	if err != nil {
		switch {
		case errors.Is(err, ErrResponseNotFound):
			response.NotFound(w, "Response not found")
		case errors.Is(err, ErrNotCastingOwner):
			response.Forbidden(w, "Only the casting owner can update response status")
		case errors.Is(err, ErrInvalidStatusTransition):
			response.BadRequest(w, "Invalid status transition")
		default:
			response.InternalError(w)
		}
		return
	}

	response.OK(w, ResponseResponseFromEntity(resp))
}

// ListMyApplications handles GET /responses/my
func (h *Handler) ListMyApplications(w http.ResponseWriter, r *http.Request) {
	// Pagination
	page := 1
	limit := 20
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	pagination := &Pagination{Page: page, Limit: limit}

	userID := middleware.GetUserID(r.Context())
	responses, total, err := h.service.ListMyApplications(r.Context(), userID, pagination)
	if err != nil {
		switch {
		case errors.Is(err, ErrProfileRequired):
			response.BadRequest(w, "You need to create a profile first")
		default:
			response.InternalError(w)
		}
		return
	}

	items := make([]*ResponseResponse, len(responses))
	for i, r := range responses {
		items[i] = ResponseResponseFromEntity(r)
	}

	response.WithMeta(w, items, response.Meta{
		Total:   total,
		Page:    page,
		Limit:   limit,
		Pages:   (total + limit - 1) / limit,
		HasNext: page*limit < total,
		HasPrev: page > 1,
	})
}
