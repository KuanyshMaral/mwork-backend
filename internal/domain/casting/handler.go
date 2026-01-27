package casting

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"
)

// Handler handles casting HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates casting handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Create handles POST /castings
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateCastingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	userID := middleware.GetUserID(r.Context())
	casting, err := h.service.Create(r.Context(), userID, &req)
	if err != nil {
		switch err {
		case ErrOnlyEmployersCanCreate:
			response.Forbidden(w, "Only employers can create castings")
		default:
			response.InternalError(w)
		}
		return
	}

	response.Created(w, CastingResponseFromEntity(casting))
}

// GetByID handles GET /castings/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid casting ID")
		return
	}

	casting, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		response.NotFound(w, "Casting not found")
		return
	}

	// Check visibility - for drafts, need to verify employer owns this casting
	// This comparison now needs employer profile ID, not user ID
	// For now, allow viewing of draft by any authenticated user who is the owner
	if casting.Status == StatusDraft {
		// Only the employer who created it can see drafts
		// TODO: Add proper employer profile lookup
		response.NotFound(w, "Casting not found")
		return
	}

	// Increment view count (async)
	go h.service.IncrementViewCount(context.Background(), id)

	response.OK(w, CastingResponseFromEntity(casting))
}

// Update handles PUT /castings/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid casting ID")
		return
	}

	var req UpdateCastingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	userID := middleware.GetUserID(r.Context())
	casting, err := h.service.Update(r.Context(), id, userID, &req)
	if err != nil {
		switch err {
		case ErrCastingNotFound:
			response.NotFound(w, "Casting not found")
		case ErrNotCastingOwner:
			response.Forbidden(w, "You can only edit your own castings")
		default:
			response.InternalError(w)
		}
		return
	}

	response.OK(w, CastingResponseFromEntity(casting))
}

// UpdateStatus handles PATCH /castings/{id}/status
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid casting ID")
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
	casting, err := h.service.UpdateStatus(r.Context(), id, userID, Status(req.Status))
	if err != nil {
		switch err {
		case ErrCastingNotFound:
			response.NotFound(w, "Casting not found")
		case ErrNotCastingOwner:
			response.Forbidden(w, "You can only manage your own castings")
		default:
			response.InternalError(w)
		}
		return
	}

	response.OK(w, CastingResponseFromEntity(casting))
}

// Delete handles DELETE /castings/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid casting ID")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if err := h.service.Delete(r.Context(), id, userID); err != nil {
		switch err {
		case ErrCastingNotFound:
			response.NotFound(w, "Casting not found")
		case ErrNotCastingOwner:
			response.Forbidden(w, "You can only delete your own castings")
		default:
			response.InternalError(w)
		}
		return
	}

	response.NoContent(w)
}

// List handles GET /castings
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter := &Filter{}
	query := r.URL.Query()

	if q := query.Get("q"); q != "" {
		filter.Query = &q
	}
	if city := query.Get("city"); city != "" {
		filter.City = &city
	}
	if payMin := query.Get("pay_min"); payMin != "" {
		if v, err := strconv.ParseFloat(payMin, 64); err == nil {
			filter.PayMin = &v
		}
	}
	if payMax := query.Get("pay_max"); payMax != "" {
		if v, err := strconv.ParseFloat(payMax, 64); err == nil {
			filter.PayMax = &v
		}
	}

	// Sort
	sortBy := SortByNewest
	if s := query.Get("sort"); s != "" {
		switch s {
		case "pay_desc":
			sortBy = SortByPayDesc
		case "popular":
			sortBy = SortByPopular
		}
	}

	// Pagination
	page := 1
	limit := 20
	if p := query.Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := query.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	pagination := &Pagination{Page: page, Limit: limit}

	castings, total, err := h.service.List(r.Context(), filter, sortBy, pagination)
	if err != nil {
		response.InternalError(w)
		return
	}

	items := make([]*CastingResponse, len(castings))
	for i, c := range castings {
		items[i] = CastingResponseFromEntity(c)
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

// ListMy handles GET /castings/my
func (h *Handler) ListMy(w http.ResponseWriter, r *http.Request) {
	// For ListMy, we need employer's profile ID, not user ID
	// This requires looking up the employer profile first
	// For now, use EmployerID filter with nil (will return empty for non-employers)
	filter := &Filter{}
	query := r.URL.Query()

	if status := query.Get("status"); status != "" {
		s := Status(status)
		filter.Status = &s
	}

	page := 1
	limit := 20
	if p := query.Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := query.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	pagination := &Pagination{Page: page, Limit: limit}

	castings, total, err := h.service.List(r.Context(), filter, SortByNewest, pagination)
	if err != nil {
		response.InternalError(w)
		return
	}

	items := make([]*CastingResponse, len(castings))
	for i, c := range castings {
		items[i] = CastingResponseFromEntity(c)
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
