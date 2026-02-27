package casting

import (
	"context"
	"encoding/json"
	"errors"
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
// @Description Создать новый кастинг. Доступно только для работодателей (Employer).
// @Tags Casting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateCastingRequest true "Данные кастинга (включая детальные требования к модели)"
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
		log.Error().
			Str("request_id", middleware.GetRequestID(r.Context())).
			Str("user_id", userID.String()).
			Err(err).
			Interface("payload", req).
			Msg("create casting failed")

		var validationErr ValidationErrors
		if errors.As(err, &validationErr) {
			response.ValidationError(w, validationErr)
			return
		}

		switch {
		case errors.Is(err, ErrOnlyEmployersCanCreate):
			response.Forbidden(w, "Only employers can create castings")
		case errors.Is(err, ErrEmployerNotVerified):
			response.Forbidden(w, "Employer account is pending verification")
		case errors.Is(err, ErrInvalidPayRange):
			response.ValidationError(w, map[string]string{"pay_min": "pay_min must be <= pay_max"})
		case errors.Is(err, ErrInvalidDateFromFormat):
			response.ValidationError(w, map[string]string{"date_from": "date_from must be RFC3339, example: 2026-05-10T10:00:00Z"})
		case errors.Is(err, ErrInvalidDateToFormat):
			response.ValidationError(w, map[string]string{"date_to": "date_to must be RFC3339, example: 2026-05-10T18:00:00Z"})
		case errors.Is(err, ErrInvalidDateRange):
			response.ValidationError(w, map[string]string{"date_from": "date_from must be <= date_to"})
		case errors.Is(err, ErrInvalidCreatorReference):
			response.ValidationError(w, map[string]string{"creator_id": "invalid creator_id"})
		case errors.Is(err, ErrDuplicateCasting):
			response.Error(w, http.StatusConflict, "CONFLICT", "duplicate title")
		case errors.Is(err, ErrCastingConstraint):
			response.ValidationError(w, map[string]string{"request": "request violates database check constraint"})
		case errors.Is(err, ErrActiveCastingQuotaExceeded):
			response.Forbidden(w, "You have reached the maximum number of active castings allowed by your plan")
		default:
			response.InternalError(w)
		}
		return
	}

	response.Created(w, CastingResponseFromEntity(casting))
}

// GetByID handles GET /castings/{id}
// @Summary Получить кастинг по ID
// @Description Получить полную информацию о кастинге. Черновики (draft) видны только владельцу.
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
// @Description Обновить данные кастинга. Доступно только владельцу.
// @Tags Casting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID кастинга"
// @Param request body UpdateCastingRequest true "Поля для обновления (включая детальные требования)"
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
		var validationErr ValidationErrors
		if errors.As(err, &validationErr) {
			response.ValidationError(w, validationErr)
			return
		}

		switch {
		case errors.Is(err, ErrCastingNotFound):
			response.NotFound(w, "Casting not found")
		case errors.Is(err, ErrNotCastingOwner):
			response.Forbidden(w, "You can only edit your own castings")
		case errors.Is(err, ErrInvalidDateFromFormat):
			response.ValidationError(w, map[string]string{"date_from": "date_from must be RFC3339, example: 2026-05-10T10:00:00Z"})
		case errors.Is(err, ErrInvalidDateToFormat):
			response.ValidationError(w, map[string]string{"date_to": "date_to must be RFC3339, example: 2026-05-10T18:00:00Z"})
		case errors.Is(err, ErrInvalidDateRange):
			response.ValidationError(w, map[string]string{"date_from": "date_from must be <= date_to"})
		case errors.Is(err, ErrInvalidPayRange):
			response.ValidationError(w, map[string]string{"pay_min": "pay_min must be <= pay_max"})
		case errors.Is(err, ErrCastingConstraint):
			response.ValidationError(w, map[string]string{"request": "request violates database check constraint"})
		default:
			response.InternalError(w)
		}
		return
	}

	response.OK(w, CastingResponseFromEntity(casting))
}

// UpdateStatus handles PATCH /castings/{id}/status
// @Summary Обновить статус кастинга
// @Description Доступные статусы: draft, active, closed.
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
		switch {
		case errors.Is(err, ErrCastingNotFound):
			response.NotFound(w, "Casting not found")
		case errors.Is(err, ErrNotCastingOwner):
			response.Forbidden(w, "You can only edit your own castings")
		case errors.Is(err, ErrInvalidStatusTransition):
			response.ValidationError(w, map[string]string{"status": "invalid status transition"})
		case errors.Is(err, ErrCastingConstraint):
			response.ValidationError(w, map[string]string{"status": "invalid status value"})
		case errors.Is(err, ErrActiveCastingQuotaExceeded):
			response.Forbidden(w, "You have reached the maximum number of active castings allowed by your plan")
		default:
			response.InternalError(w)
		}
		return
	}

	response.OK(w, CastingResponseFromEntity(casting))
}

// Delete handles DELETE /castings/{id}
// @Summary Удалить кастинг
// @Description Удаляет кастинг. Только владелец может удалить.
// @Tags Casting
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID кастинга"
// @Success 204 "No Content"
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
// @Summary Список кастингов (Поиск)
// @Description Поиск кастингов по фильтрам.
// @Tags Casting
// @Produce json
// @Param city query string false "Город"
// @Param q query string false "Поисковый запрос (по заголовку)"
// @Param status query string false "Статус (active, closed, draft)"
// @Param creator_id query string false "ID создателя"
// @Param pay_min query number false "Минимальная оплата"
// @Param pay_max query number false "Максимальная оплата"
// @Param sort_by query string false "Сортировка (newest, pay_desc, pay_asc, views)"
// @Param page query int false "Номер страницы" default(1)
// @Param limit query int false "Количество на странице" default(20)
// @Success 200 {object} response.Response{data=[]CastingResponse,meta=response.Meta}
// @Failure 500 {object} response.Response
// @Router /castings [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := &Filter{}

	if city := q.Get("city"); city != "" {
		filter.City = &city
	}
	if query := q.Get("q"); query != "" {
		filter.Query = &query
	}
	if status := q.Get("status"); status != "" {
		s := Status(status)
		switch s {
		case StatusDraft, StatusActive, StatusClosed:
			filter.Status = &s
		default:
			response.ValidationError(w, map[string]string{"status": "status must be one of: draft, active, closed"})
			return
		}
	}
	if creatorID := q.Get("creator_id"); creatorID != "" {
		if id, err := uuid.Parse(creatorID); err == nil {
			filter.CreatorID = &id
		}
	}
	if payMin := q.Get("pay_min"); payMin != "" {
		if v, err := strconv.ParseFloat(payMin, 64); err == nil {
			filter.PayMin = &v
		}
	}
	if payMax := q.Get("pay_max"); payMax != "" {
		if v, err := strconv.ParseFloat(payMax, 64); err == nil {
			filter.PayMax = &v
		}
	}
	if tags, ok := q["tags"]; ok && len(tags) > 0 {
		filter.Tags = tags
	}

	sortBy := SortBy(q.Get("sort_by"))
	if sortBy == "" {
		sortBy = SortByNewest
	}

	page := 1
	if p := q.Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	limit := 20
	if l := q.Get("limit"); l != "" {
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

	items := make([]*CastingResponse, 0, len(castings))
	for _, c := range castings {
		items = append(items, CastingResponseFromEntity(c))
	}

	pages := total / limit
	if total%limit != 0 {
		pages++
	}

	response.WithMeta(w, items, response.Meta{
		Total:   total,
		Page:    page,
		Limit:   limit,
		Pages:   pages,
		HasNext: page < pages,
		HasPrev: page > 1,
	})
}

// ListMy handles GET /castings/my
// @Summary Мои кастинги
// @Description Получить список кастингов текущего пользователя (работодателя).
// @Tags Casting
// @Produce json
// @Security BearerAuth
// @Param page query int false "Номер страницы" default(1)
// @Param limit query int false "Количество на странице" default(20)
// @Success 200 {object} response.Response{data=[]CastingResponse,meta=response.Meta}
// @Failure 500 {object} response.Response
// @Router /castings/my [get]
func (h *Handler) ListMy(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	q := r.URL.Query()
	page := 1
	if p := q.Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	limit := 20
	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	pagination := &Pagination{Page: page, Limit: limit}

	castings, total, err := h.service.ListByCreator(r.Context(), userID, pagination)
	if err != nil {
		response.InternalError(w)
		return
	}

	items := make([]*CastingResponse, 0, len(castings))
	for _, c := range castings {
		items = append(items, CastingResponseFromEntity(c))
	}

	pages := total / limit
	if total%limit != 0 {
		pages++
	}

	response.WithMeta(w, items, response.Meta{
		Total:   total,
		Page:    page,
		Limit:   limit,
		Pages:   pages,
		HasNext: page < pages,
		HasPrev: page > 1,
	})
}
