package moderation

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"
)

// Handler handles moderation HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates moderation handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// BlockUser blocks a user
// @Summary Заблокировать пользователя
// @Tags Moderation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body BlockUserRequest true "ID пользователя для блокировки"
// @Success 200 {object} response.Response
// @Failure 400,401,500 {object} response.Response
// @Router /moderation/block [post]
func (h *Handler) BlockUser(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req BlockUserRequest
	if err := response.DecodeJSON(r.Body, &req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	if err := h.service.BlockUser(r.Context(), userID, &req); err != nil {
		switch err {
		case ErrCannotBlockSelf:
			response.BadRequest(w, "Cannot block yourself")
		case ErrAlreadyBlocked:
			response.BadRequest(w, "User already blocked")
		default:
			response.InternalError(w)
		}
		return
	}

	response.OK(w, map[string]string{"message": "User blocked successfully"})
}

// UnblockUser unblocks a user
// @Summary Разблокировать пользователя
// @Tags Moderation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body BlockUserRequest true "ID пользователя для разблокировки"
// @Success 200 {object} response.Response
// @Failure 400,401,404,500 {object} response.Response
// @Router /moderation/block [delete]
func (h *Handler) UnblockUser(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req BlockUserRequest
	if err := response.DecodeJSON(r.Body, &req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	if err := h.service.UnblockUser(r.Context(), userID, req.UserID); err != nil {
		if err == ErrBlockNotFound {
			response.NotFound(w, "Block not found")
		} else {
			response.InternalError(w)
		}
		return
	}

	response.OK(w, map[string]string{"message": "User unblocked successfully"})
}

// @Summary Список заблокированных пользователей
// @Tags Moderation
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 401,500 {object} response.Response
// @Router /moderation/blocks [get]
// ListBlocks lists blocked users
// GET /moderation/blocks
func (h *Handler) ListBlocks(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	blocks, err := h.service.ListBlocks(r.Context(), userID)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, blocks)
}

// @Summary Создать жалобу
// @Tags Moderation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateReportRequest true "Данные жалобы"
// @Success 201 {object} response.Response
// @Failure 400,401,404,422,500 {object} response.Response
// @Router /moderation/report [post]
// CreateReport creates a new report
// POST /moderation/report
func (h *Handler) CreateReport(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req CreateReportRequest
	if err := response.DecodeJSON(r.Body, &req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	report, err := h.service.CreateReport(r.Context(), userID, &req)
	if err != nil {
		switch err {
		case ErrCannotReportSelf:
			response.BadRequest(w, "Cannot report yourself")
		case ErrReportNotFound:
			response.NotFound(w, "Reported user not found")
		default:
			response.InternalError(w)
		}
		return
	}

	response.Created(w, report)
}

// ListMyReports lists reports created by current user
// @Summary Мои жалобы
// @Tags Moderation
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 401,500 {object} response.Response
// @Router /moderation/reports/mine [get]
func (h *Handler) ListMyReports(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	reports, err := h.service.ListMyReports(r.Context(), userID)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, reports)
}

// ListMyReports lists reports created by current user
// @Summary Все жалобы (админ)
// @Tags Moderation Admin
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Лимит"
// @Param offset query int false "Смещение"
// @Param status query string false "Статус (pending|resolved|rejected)"
// @Success 200 {object} response.Response
// @Failure 400,401,403,500 {object} response.Response
// @Router /admin/reports [get]ports/mine
// ListReports lists all reports (admin only)
// GET /admin/reports
func (h *Handler) ListReports(w http.ResponseWriter, r *http.Request) {
	// Parse query params
	filter := &ListReportsFilter{
		Limit:  50,
		Offset: 0,
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = ReportStatus(status)
	}

	reports, err := h.service.ListReports(r.Context(), filter)
	if err != nil {
		response.InternalError(w)
		return
	}

	// Get total count
	total, _ := h.service.CountReports(r.Context(), filter)

	meta := response.Meta{
		Total: total,
		Limit: filter.Limit,
	}

	response.WithMeta(w, reports, meta)
}

// ResolveReport resolves a report (admin only)
// @Summary Разрешить жалобу (админ)
// @Tags Moderation Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID жалобы"
// @Param request body ResolveReportRequest true "Решение жалобы"
// @Success 200 {object} response.Response
// @Failure 400,401,403,404,422,500 {object} response.Response
// @Router /admin/reports/{id}/resolve [post]
func (h *Handler) ResolveReport(w http.ResponseWriter, r *http.Request) {
	reportID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid report ID")
		return
	}

	var req ResolveReportRequest
	if err := response.DecodeJSON(r.Body, &req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	if err := h.service.ResolveReport(r.Context(), reportID, &req); err != nil {
		switch err {
		case ErrReportNotFound:
			response.NotFound(w, "Report not found")
		case ErrInvalidReportStatus:
			response.BadRequest(w, "Invalid action")
		default:
			response.InternalError(w)
		}
		return
	}

	response.OK(w, map[string]string{"message": "Report resolved successfully"})
}
