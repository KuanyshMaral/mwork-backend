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
// @Summary Создание профиля модели
// @Description Создает профиль модели для текущего пользователя с ролью model.
// @Tags Profiles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateModelProfileRequest true "Данные профиля модели"
// @Success 201 {object} response.Response{data=ModelProfileResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 409 {object} response.Response
// @Failure 422 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /profiles/model [post]
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
// @Summary Создание профиля работодателя
// @Description Создает профиль работодателя для текущего пользователя с ролью employer.
// @Tags Profiles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateEmployerProfileRequest true "Данные профиля работодателя"
// @Success 201 {object} response.Response{data=EmployerProfileResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 409 {object} response.Response
// @Failure 422 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /profiles/employer [post]
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
// @Summary Мой профиль
// @Description Возвращает профиль текущего пользователя (модель или работодатель).
// @Tags Profiles
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /profiles/me [get]
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
// @Summary Получение профиля модели
// @Description Возвращает публичный профиль модели по идентификатору.
// @Tags Profiles
// @Produce json
// @Param id path string true "ID профиля"
// @Success 200 {object} response.Response{data=ModelProfileResponse}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /profiles/models/{id} [get]
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
// @Summary Обновление профиля модели
// @Description Обновляет профиль модели. Доступно только владельцу профиля.
// @Tags Profiles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID профиля"
// @Param request body UpdateModelProfileRequest true "Обновление профиля модели"
// @Success 200 {object} response.Response{data=ModelProfileResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 422 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /profiles/models/{id} [put]
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
// @Summary Список моделей
// @Description Возвращает список профилей моделей с фильтрами и пагинацией.
// @Tags Profiles
// @Produce json
// @Param q query string false "Поиск"
// @Param city query string false "Город"
// @Param gender query string false "Пол"
// @Param age_min query int false "Минимальный возраст"
// @Param age_max query int false "Максимальный возраст"
// @Param height_min query number false "Минимальный рост"
// @Param height_max query number false "Максимальный рост"
// @Param page query int false "Страница"
// @Param limit query int false "Лимит"
// @Success 200 {object} response.Response{data=[]ModelProfileResponse}
// @Failure 500 {object} response.Response
// @Router /profiles/models [get]
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

// ListPromotedModels handles GET /profiles/models/promoted
// @Summary Список продвигаемых моделей
// @Description Возвращает список продвигаемых профилей моделей.
// @Tags Profiles
// @Produce json
// @Param city query string false "Город"
// @Param limit query int false "Лимит"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /profiles/models/promoted [get]
func (h *Handler) ListPromotedModels(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	var city *string
	if c := query.Get("city"); c != "" {
		city = &c
	}

	limit := 20
	if l := query.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	profiles, err := h.service.ListPromotedModels(r.Context(), city, limit)
	if err != nil {
		response.InternalError(w)
		return
	}

	items := make([]*ModelProfileResponse, len(profiles))
	for i, p := range profiles {
		items[i] = ModelProfileResponseFromEntity(p)
	}

	response.OK(w, map[string]interface{}{
		"items": items,
		"total": len(items),
	})
}
