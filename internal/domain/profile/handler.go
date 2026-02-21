package profile

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/errorhandler"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"

	attachmentDomain "github.com/mwork/mwork-api/internal/domain/attachment"
)

// Handler handles profile HTTP requests
type Handler struct {
	service           *Service
	attachmentService *attachmentDomain.Service
}

// NewHandler creates profile handler
func NewHandler(service *Service, attachmentService *attachmentDomain.Service) *Handler {
	return &Handler{service: service, attachmentService: attachmentService}
}

// GetMe handles GET /profiles/me
// @Summary Мой профиль
// @Description Возвращает профиль текущего пользователя (модель или работодатель).
// @Tags Profile
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /profiles/me [get]
// Returns either a ModelProfileResponse or EmployerProfileResponse depending on user type
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	// Fetch user to get credit balance
	user, err := h.service.userRepo.GetByID(r.Context(), userID)
	creditBalance := 0
	if err == nil && user != nil {
		creditBalance = user.CreditBalance
	}

	// Try to get model profile first
	modelProfile, err := h.service.GetModelProfileByUserID(r.Context(), userID)
	if err == nil && modelProfile != nil {
		resp := ModelProfileResponseFromEntity(modelProfile)
		resp.CreditBalance = creditBalance

		// Fetch portfolio securely
		if h.attachmentService != nil {
			attachments, err := h.attachmentService.ListByTarget(r.Context(), attachmentDomain.TargetModelPortfolio, modelProfile.ID)
			if err == nil {
				resp.Portfolio = make([]attachmentDomain.AttachmentWithURL, len(attachments))
				for i, a := range attachments {
					resp.Portfolio[i] = *a
				}
			}
		}

		response.OK(w, resp)
		return
	}

	// Try employer profile
	employerProfile, err := h.service.GetEmployerProfileByUserID(r.Context(), userID)
	if err == nil && employerProfile != nil {
		resp := EmployerProfileResponseFromEntity(employerProfile)
		resp.CreditBalance = creditBalance
		response.OK(w, resp)
		return
	}

	adminProfile, err := h.service.GetAdminProfileByUserID(r.Context(), userID)
	if err == nil && adminProfile != nil {
		response.OK(w, AdminProfileResponseFromEntity(adminProfile))
		return
	}

	response.NotFound(w, "Profile not found")
}

// GetModelByID handles GET /profiles/models/{id}
// @Summary Получение профиля модели
// @Description Возвращает публичный профиль модели по идентификатору.
// @Tags Profile
// @Produce json
// @Param id path string true "ID профиля"
// @Success 200 {object} response.Response{data=ModelProfileResponse}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /profiles/models/{id} [get]
func (h *Handler) GetModelByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	profile, err := h.service.GetModelProfileByID(r.Context(), id)
	if err != nil {
		response.NotFound(w, "Profile not found")
		return
	}

	go h.service.IncrementModelViewCount(context.Background(), id)

	resp := ModelProfileResponseFromEntity(profile)
	if h.attachmentService != nil {
		attachments, err := h.attachmentService.ListByTarget(r.Context(), attachmentDomain.TargetModelPortfolio, profile.ID)
		if err == nil {
			resp.Portfolio = make([]attachmentDomain.AttachmentWithURL, len(attachments))
			for i, a := range attachments {
				resp.Portfolio[i] = *a
			}
		}
	}

	response.OK(w, resp)
}

// UpdateModel handles PUT /profiles/models/{userId}
// @Summary Обновление профиля модели
// @Description Обновляет профиль модели. Доступно только владельцу профиля.
// @Tags Profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path string true "ID пользователя"
// @Param request body UpdateModelProfileRequest true "Обновление профиля модели"
// @Success 200 {object} response.Response{data=ModelProfileResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 422 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /profiles/models/{userId} [put]
func (h *Handler) UpdateModel(w http.ResponseWriter, r *http.Request) {
	userIDParam, err := uuid.Parse(chi.URLParam(r, "userId"))
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

	callerID := middleware.GetUserID(r.Context())
	if userIDParam != callerID {
		response.Forbidden(w, "You can only edit your own profile")
		return
	}

	profile, err := h.service.UpdateModelProfile(r.Context(), callerID, &req)
	if err != nil {
		switch err {
		case ErrProfileNotFound:
			response.NotFound(w, "Profile not found")
		case ErrNotProfileOwner:
			response.Forbidden(w, "You can only edit your own profile")
		default:
			errorhandler.HandleError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", err)
		}
		return
	}

	response.OK(w, ModelProfileResponseFromEntity(profile))
}

// UpdateEmployer handles PUT /profiles/employers/{userId}
// @Summary Обновление профиля работодателя
// @Description Обновляет профиль работодателя. Доступно только владельцу профиля.
// @Tags Profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path string true "ID пользователя"
// @Param request body UpdateEmployerProfileRequest true "Обновление профиля работодателя"
// @Success 200 {object} response.Response{data=EmployerProfileResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 422 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /profiles/employers/{userId} [put]
func (h *Handler) UpdateEmployer(w http.ResponseWriter, r *http.Request) {
	userIDParam, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	var req UpdateEmployerProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	callerID := middleware.GetUserID(r.Context())
	if userIDParam != callerID {
		response.Forbidden(w, "You can only edit your own profile")
		return
	}

	profile, err := h.service.UpdateEmployerProfile(r.Context(), callerID, &req)
	if err != nil {
		switch err {
		case ErrProfileNotFound:
			response.NotFound(w, "Profile not found")
		case ErrNotProfileOwner:
			response.Forbidden(w, "You can only edit your own profile")
		default:
			errorhandler.HandleError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", err)
		}
		return
	}

	response.OK(w, EmployerProfileResponseFromEntity(profile))
}

// ListModels handles GET /profiles/models
// @Summary Список моделей
// @Description Возвращает список профилей моделей с фильтрами и пагинацией.
// @Tags Profile
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
		errorhandler.HandleError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", err)
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
// @Tags Profile
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
		errorhandler.HandleError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", err)
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

// UpdateAdmin handles PUT /profiles/admins/{userId}
// @Summary Обновление профиля администратора
// @Description Обновляет профиль администратора. Доступно только владельцу профиля.
// @Tags Profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path string true "ID пользователя"
// @Param request body UpdateAdminProfileRequest true "Обновление профиля администратора"
// @Success 200 {object} response.Response{data=AdminProfileResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 422 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /profiles/admins/{userId} [put]
func (h *Handler) UpdateAdmin(w http.ResponseWriter, r *http.Request) {
	userIDParam, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	callerID := middleware.GetUserID(r.Context())
	if userIDParam != callerID {
		response.Forbidden(w, "You can only edit your own profile")
		return
	}

	var req UpdateAdminProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}
	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	profile, err := h.service.UpdateAdminProfileByUserID(r.Context(), callerID, &req)
	if err != nil {
		switch err {
		case ErrProfileNotFound:
			response.NotFound(w, "Profile not found")
		default:
			errorhandler.HandleError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", err)
		}
		return
	}

	response.OK(w, AdminProfileResponseFromEntity(profile))
}
