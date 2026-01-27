package casting

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"
)

// Handler handles casting HTTP requests
type Handler struct {
	service        *Service
	profileService ProfileService
}

// ProfileService interface for profile operations
type ProfileService interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (EmployerProfileInfo, error)
}

// EmployerProfileInfo interface for employer profile data
type EmployerProfileInfo interface {
	GetID() uuid.UUID
}

// NewHandler creates casting handler
func NewHandler(service *Service, profileService ProfileService) *Handler {
	return &Handler{
		service:        service,
		profileService: profileService,
	}
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
	ctx := r.Context()

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid casting ID")
		return
	}

	casting, err := h.service.GetByID(ctx, id)
	if err != nil {
		response.NotFound(w, "Casting not found")
		return
	}

	// Check visibility - for drafts, verify employer owns this casting
	if casting.Status == StatusDraft {
		// Get current user ID
		userID := middleware.GetUserID(ctx)
		if userID == uuid.Nil {
			response.Error(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		// Get employer profile
		userProfile, err := h.profileService.GetByUserID(ctx, userID)
		if err != nil {
			log.Warn().
				Err(err).
				Str("user_id", userID.String()).
				Msg("Failed to get profile for draft access check")
			response.Error(w, http.StatusForbidden, "no profile found")
			return
		}

		// Check ownership
		if casting.CreatorID != userProfile.GetID() {
			log.Warn().
				Str("casting_id", id.String()).
				Str("user_id", userID.String()).
				Str("creator_id", casting.CreatorID.String()).
				Msg("Unauthorized draft access attempt")
			response.Error(w, http.StatusForbidden, "not authorized to view draft")
			return
		}
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
	ctx := r.Context()

	// Get current user ID
	userID := middleware.GetUserID(ctx)
	if userID == uuid.Nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get employer profile to filter by creator_id
	profile, err := h.profileService.GetByUserID(ctx, userID)
	if err == sql.ErrNoRows || err != nil {
		// New employer, no castings yet - return empty list
		log.Info().Str("user_id", userID.String()).Msg("No profile found, returning empty castings list")
		response.JSON(w, http.StatusOK, map[string]interface{}{
			"data": []interface{}{},
			"meta": response.Meta{
				Total:   0,
				Page:    1,
				Limit:   20,
				Pages:   0,
				HasNext: false,
				HasPrev: false,
			},
		})
		return
	}

	// Parse filter params
	profileID := profile.GetID()
	filter := &Filter{
		EmployerID: &profileID, // Filter by employer profile ID
	}

	query := r.URL.Query()
	if status := query.Get("status"); status != "" {
		s := Status(status)
		filter.Status = &s
	}

	// Parse pagination
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

	// List castings by creator
	castings, total, err := h.service.List(ctx, filter, SortByNewest, pagination)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch my castings")
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
