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

// ProfileService defines profile operations needed by casting
type ProfileService interface {
	GetEmployerProfileByUserID(ctx context.Context, userID uuid.UUID) (*EmployerProfile, error)
}

// EmployerProfile represents an employer profile entity
type EmployerProfile struct {
	ID     uuid.UUID
	UserID uuid.UUID
}

// NewHandler creates casting handler
func NewHandler(service *Service, profileService ProfileService) *Handler {
	return &Handler{
		service:        service,
		profileService: profileService,
	}
}

// Create handles POST /castings
// @Summary Создать кастинг
// @Tags Casting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateCastingRequest true "Данные кастинга"
// @Success 201 {object} response.Response{data=CastingResponse}
// @Failure 400,403,422,500 {object} response.Response
// @Router /castings [post]
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
// @Summary Получить кастинг по ID
// @Tags Casting
// @Produce json
// @Param id path string true "ID кастинга"
// @Success 200 {object} response.Response{data=CastingResponse}
// @Failure 400,403,404,500 {object} response.Response
// @Router /castings/{id} [get]
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

	// Check visibility - for drafts, need to verify employer owns this casting
	if casting.Status == StatusDraft {
		// Get userID from context
		userID := middleware.GetUserID(ctx)
		if userID == uuid.Nil {
			// No user authenticated, can't view draft
			response.NotFound(w, "Casting not found")
			return
		}

		// Get employer profile to check ownership
		userProfile, err := h.profileService.GetEmployerProfileByUserID(ctx, userID)
		if err != nil {
			// No profile found or error
			log.Warn().
				Str("user_id", userID.String()).
				Msg("no profile found for draft access attempt")
			response.Forbidden(w, "no profile found")
			return
		}

		// Check if user owns this casting
		if casting.CreatorID != userProfile.ID {
			log.Warn().
				Str("casting_id", id.String()).
				Str("user_id", userID.String()).
				Str("creator_id", casting.CreatorID.String()).
				Msg("unauthorized draft access attempt")
			response.Forbidden(w, "not authorized to view draft")
			return
		}
	}

	// Increment view count (async)
	go h.service.IncrementViewCount(context.Background(), id)

	response.OK(w, CastingResponseFromEntity(casting))
}

// Update handles PUT /castings/{id}
// @Summary Обновить кастинг
// @Tags Casting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID кастинга"
// @Param request body UpdateCastingRequest true "Поля для обновления"
// @Success 200 {object} response.Response{data=CastingResponse}
// @Failure 400,403,404,422,500 {object} response.Response
// @Router /castings/{id} [put]
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
// @Summary Обновить статус кастинга
// @Tags Casting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID кастинга"
// @Param request body UpdateStatusRequest true "Новый статус"
// @Success 200 {object} response.Response{data=CastingResponse}
// @Failure 400,403,404,422,500 {object} response.Response
// @Router /castings/{id}/status [patch]
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
// @Summary Удалить кастинг
// @Tags Casting
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID кастинга"
// @Success 204 {string} string "No Content"
// @Failure 400,403,404,500 {object} response.Response
// @Router /castings/{id} [delete]
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
// @Summary Список кастингов
// @Tags Casting
// @Produce json
// @Param page query int false "Страница"
// @Param limit query int false "Лимит"
// @Param city query string false "Город"
// @Param status query string false "Статус"
// @Success 200 {object} response.Response{data=[]CastingResponse}
// @Failure 500 {object} response.Response
// @Router /castings [get]
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
// @Summary Мои кастинги
// @Tags Casting
// @Produce json
// @Security BearerAuth
// @Param page query int false "Страница"
// @Param limit query int false "Лимит"
// @Param status query string false "Статус"
// @Success 200 {object} response.Response{data=[]CastingResponse}
// @Failure 500 {object} response.Response
// @Router /castings/my [get]
func (h *Handler) ListMy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get current user
	userID := middleware.GetUserID(ctx)
	if userID == uuid.Nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	// Get employer profile to filter by creator_id
	profile, err := h.profileService.GetEmployerProfileByUserID(ctx, userID)
	if err != nil {
		// If user has no profile yet, return empty list
		if err == sql.ErrNoRows {
			response.JSON(w, http.StatusOK, map[string]interface{}{
				"success": true,
				"data":    []interface{}{},
				"meta": map[string]interface{}{
					"total":    0,
					"page":     1,
					"limit":    20,
					"pages":    0,
					"has_next": false,
					"has_prev": false,
				},
			})
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get profile")
		return
	}

	// Build filter with employer profile ID
	filter := &Filter{
		CreatorID: &profile.ID,
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

	castings, total, err := h.service.List(ctx, filter, SortByNewest, pagination)
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
