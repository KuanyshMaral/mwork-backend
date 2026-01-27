package profile

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

// Handler handles profile HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates profile handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// CreateModel handles POST /profiles/model
func (h *Handler) CreateModel(w http.ResponseWriter, r *http.Request) {
	var req CreateModelProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	userID := middleware.GetUserID(r.Context())
	profile, err := h.service.CreateModelProfile(r.Context(), userID, &req)
	if err != nil {
		switch err {
		case ErrProfileAlreadyExists:
			response.Conflict(w, "Profile already exists for this user")
		case ErrInvalidProfileType:
			response.BadRequest(w, "User role must be model")
		default:
			response.InternalError(w)
		}
		return
	}

	response.Created(w, ModelProfileResponseFromEntity(profile))
}

// CreateEmployer handles POST /profiles/employer
func (h *Handler) CreateEmployer(w http.ResponseWriter, r *http.Request) {
	var req CreateEmployerProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	userID := middleware.GetUserID(r.Context())
	profile, err := h.service.CreateEmployerProfile(r.Context(), userID, &req)
	if err != nil {
		switch err {
		case ErrProfileAlreadyExists:
			response.Conflict(w, "Profile already exists for this user")
		case ErrInvalidProfileType:
			response.BadRequest(w, "User role must be employer")
		default:
			response.InternalError(w)
		}
		return
	}

	response.Created(w, EmployerProfileResponseFromEntity(profile))
}

// GetMe handles GET /profiles/me
// Returns either a ModelProfileResponse or EmployerProfileResponse depending on user type
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	// Try to get model profile first
	modelProfile, err := h.service.GetModelProfileByUserID(r.Context(), userID)
	if err == nil && modelProfile != nil {
		response.OK(w, ModelProfileResponseFromEntity(modelProfile))
		return
	}

	// Try employer profile
	employerProfile, err := h.service.GetEmployerProfileByUserID(r.Context(), userID)
	if err == nil && employerProfile != nil {
		response.OK(w, EmployerProfileResponseFromEntity(employerProfile))
		return
	}

	response.NotFound(w, "Profile not found. Please create a profile first.")
}

// GetModelByID handles GET /profiles/models/{id}
func (h *Handler) GetModelByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid profile ID")
		return
	}

	profile, err := h.service.GetModelProfileByID(r.Context(), id)
	if err != nil {
		response.NotFound(w, "Profile not found")
		return
	}

	// Increment view count (async)
	go h.service.IncrementModelViewCount(context.Background(), id)

	response.OK(w, ModelProfileResponseFromEntity(profile))
}

// UpdateModel handles PUT /profiles/models/{id}
func (h *Handler) UpdateModel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid profile ID")
		return
	}

	var req UpdateModelProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	userID := middleware.GetUserID(r.Context())
	profile, err := h.service.UpdateModelProfile(r.Context(), id, userID, &req)
	if err != nil {
		switch err {
		case ErrProfileNotFound:
			response.NotFound(w, "Profile not found")
		case ErrNotProfileOwner:
			response.Forbidden(w, "You can only edit your own profile")
		default:
			response.InternalError(w)
		}
		return
	}

	response.OK(w, ModelProfileResponseFromEntity(profile))
}

// ListModels handles GET /profiles/models
func (h *Handler) ListModels(w http.ResponseWriter, r *http.Request) {
	filter := &Filter{}
	query := r.URL.Query()

	if q := query.Get("q"); q != "" {
		filter.Query = &q
	}
	if city := query.Get("city"); city != "" {
		filter.City = &city
	}
	if gender := query.Get("gender"); gender != "" {
		filter.Gender = &gender
	}
	if ageMin := query.Get("age_min"); ageMin != "" {
		if v, err := strconv.Atoi(ageMin); err == nil {
			filter.AgeMin = &v
		}
	}
	if ageMax := query.Get("age_max"); ageMax != "" {
		if v, err := strconv.Atoi(ageMax); err == nil {
			filter.AgeMax = &v
		}
	}
	if heightMin := query.Get("height_min"); heightMin != "" {
		if v, err := strconv.ParseFloat(heightMin, 64); err == nil {
			filter.HeightMin = &v
		}
	}
	if heightMax := query.Get("height_max"); heightMax != "" {
		if v, err := strconv.ParseFloat(heightMax, 64); err == nil {
			filter.HeightMax = &v
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

	profiles, total, err := h.service.ListModels(r.Context(), filter, pagination)
	if err != nil {
		response.InternalError(w)
		return
	}

	items := make([]*ModelProfileResponse, len(profiles))
	for i, p := range profiles {
		items[i] = ModelProfileResponseFromEntity(p)
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
