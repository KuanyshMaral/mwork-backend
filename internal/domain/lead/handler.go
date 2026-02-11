package lead

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"
)

// Handler handles lead HTTP requests
type Handler struct {
	svc *Service
}

// NewHandler creates lead handler
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// SubmitLead handles POST /leads/employer (public)
// @Summary Создание лида работодателя
// @Description Публичная форма создания лида работодателя.
// @Tags Leads
// @Accept json
// @Produce json
// @Param request body CreateLeadRequest true "Данные лида"
// @Success 201 {object} response.Response{data=LeadSubmittedResponse}
// @Failure 400 {object} response.Response
// @Failure 422 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /leads/employer [post]
func (h *Handler) SubmitLead(w http.ResponseWriter, r *http.Request) {
	var req CreateLeadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	// Get UTM from query params if not in body
	if req.UTMSource == "" {
		req.UTMSource = r.URL.Query().Get("utm_source")
	}
	if req.UTMMedium == "" {
		req.UTMMedium = r.URL.Query().Get("utm_medium")
	}
	if req.UTMCampaign == "" {
		req.UTMCampaign = r.URL.Query().Get("utm_campaign")
	}

	// Get client IP
	ip := net.ParseIP(r.Header.Get("X-Forwarded-For"))
	if ip == nil {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		ip = net.ParseIP(host)
	}

	lead, err := h.svc.SubmitLead(r.Context(), &req, ip, r.UserAgent())
	if err != nil {
		response.InternalError(w)
		return
	}

	response.Created(w, &LeadSubmittedResponse{
		LeadID:  lead.ID,
		Message: "Спасибо за заявку! Наш менеджер свяжется с вами в течение 24 часов.",
	})
}

// List handles GET /admin/leads
// @Summary Список лидов
// @Description Возвращает список лидов для админской части.
// @Tags Admin Leads
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Лимит"
// @Param offset query int false "Смещение"
// @Param status query string false "Статус"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/leads [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	var status *Status
	if s := r.URL.Query().Get("status"); s != "" {
		st := Status(s)
		status = &st
	}

	leads, total, err := h.svc.ListLeads(r.Context(), status, limit, offset)
	if err != nil {
		response.InternalError(w)
		return
	}

	items := make([]*LeadResponse, len(leads))
	for i, lead := range leads {
		items[i] = ToResponse(lead)
	}

	response.OK(w, map[string]interface{}{
		"items": items,
		"total": total,
	})
}

// GetByID handles GET /admin/leads/{id}
// @Summary Лид по ID
// @Description Возвращает лид по идентификатору.
// @Tags Admin Leads
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID лида"
// @Success 200 {object} response.Response{data=LeadResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/leads/{id} [get]
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid lead ID")
		return
	}

	lead, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if err == ErrLeadNotFound {
			response.NotFound(w, "Lead not found")
			return
		}
		response.InternalError(w)
		return
	}

	response.OK(w, ToResponse(lead))
}

// UpdateStatus handles PATCH /admin/leads/{id}/status
// @Summary Обновление статуса лида
// @Description Обновляет статус лида в админской части.
// @Tags Admin Leads
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID лида"
// @Param request body UpdateStatusRequest true "Новый статус"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 422 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/leads/{id}/status [patch]
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid lead ID")
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

	if err := h.svc.UpdateStatus(r.Context(), id, Status(req.Status), req.Notes, req.RejectionReason); err != nil {
		if err == ErrLeadNotFound {
			response.NotFound(w, "Lead not found")
			return
		}
		if err == ErrAlreadyConverted {
			response.BadRequest(w, "Lead is already converted")
			return
		}
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]string{"status": "updated"})
}

// MarkContacted handles POST /admin/leads/{id}/contacted
// @Summary Отметить лид как contacted
// @Description Обновляет лид как контактированный.
// @Tags Admin Leads
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID лида"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/leads/{id}/contacted [post]
func (h *Handler) MarkContacted(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid lead ID")
		return
	}

	if err := h.svc.MarkContacted(r.Context(), id); err != nil {
		if err == ErrLeadNotFound {
			response.NotFound(w, "Lead not found")
			return
		}
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]string{"status": "marked"})
}

// Assign handles POST /admin/leads/{id}/assign
// @Summary Назначить ответственного по лиду
// @Description Назначает администратора и приоритет для лида.
// @Tags Admin Leads
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID лида"
// @Param request body AssignRequest true "Назначение"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/leads/{id}/assign [post]
func (h *Handler) Assign(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid lead ID")
		return
	}

	var req AssignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	adminID, err := uuid.Parse(req.AdminID)
	if err != nil {
		response.BadRequest(w, "Invalid admin ID")
		return
	}

	if err := h.svc.Assign(r.Context(), id, adminID, req.Priority); err != nil {
		if err == ErrLeadNotFound {
			response.NotFound(w, "Lead not found")
			return
		}
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]string{"status": "assigned"})
}

// Convert handles POST /admin/leads/{id}/convert
// @Summary Конвертация лида в работодателя
// @Description Создает пользователя и организацию на основе лида.
// @Tags Admin Leads
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID лида"
// @Param request body ConvertRequest true "Данные конвертации"
// @Success 201 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 422 {object} response.Response
// @Router /admin/leads/{id}/convert [post]
func (h *Handler) Convert(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid lead ID")
		return
	}

	var req ConvertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	user, org, err := h.svc.Convert(r.Context(), id, &req)
	if err != nil {
		switch err {
		case ErrLeadNotFound:
			response.NotFound(w, "Lead not found")
		case ErrAlreadyConverted:
			response.BadRequest(w, "Lead is already converted")
		case ErrCannotConvert:
			response.BadRequest(w, "Lead must be qualified or contacted to convert")
		default:
			response.BadRequest(w, err.Error())
		}
		return
	}

	response.Created(w, map[string]interface{}{
		"user_id":         user.ID,
		"organization_id": org.ID,
		"email":           user.Email,
		"message":         "Employer account created successfully",
	})
}

// Stats handles GET /admin/leads/stats
// @Summary Статистика лидов
// @Description Возвращает агрегированную статистику по лидам для админской части.
// @Tags Admin Leads
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/leads/stats [get]
func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetStats(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, stats)
}

// Reject handles POST /admin/leads/{id}/reject
// @Summary Отклонение лида
// @Description Устанавливает статус rejected для лида с причиной.
// @Tags Admin Leads
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID лида"
// @Param request body RejectLeadRequest true "Причина отклонения"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/leads/{id}/reject [post]
func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid lead ID")
		return
	}

	var req RejectLeadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if req.Reason == "" {
		response.BadRequest(w, "Rejection reason is required")
		return
	}

	if err := h.svc.UpdateStatus(r.Context(), id, StatusRejected, "", req.Reason); err != nil {
		if err == ErrLeadNotFound {
			response.NotFound(w, "Lead not found")
			return
		}
		if err == ErrAlreadyConverted {
			response.BadRequest(w, "Lead is already converted")
			return
		}
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]string{"status": "rejected"})
}
