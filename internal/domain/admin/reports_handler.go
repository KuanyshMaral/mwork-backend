package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"
)

// ListReports returns paginated reports with filters
// @Summary Список репортов
// @Tags Admin Reports
// @Produce json
// @Security BearerAuth
// @Param page query int false "Страница"
// @Param limit query int false "Лимит"
// @Param status query string false "Статус"
// @Success 200 {object} response.Response{data=ListReportsResponse}
// @Failure 500 {object} response.Response
// @Router /admin/reports [get]
func (h *Handler) ListReports(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Parse status filter
	var statusFilter *string
	if status := r.URL.Query().Get("status"); status != "" {
		statusFilter = &status
	}

	// Get reports from service
	reports, total, err := h.service.ListReports(r.Context(), page, limit, statusFilter)
	if err != nil {
		response.InternalError(w)
		return
	}

	// Convert to response format
	reportResponses := make([]ReportResponse, len(reports))
	for i, report := range reports {
		reportResponses[i] = ReportResponseFromEntity(report)
	}

	// Return paginated response
	response.OK(w, &ListReportsResponse{
		Reports: reportResponses,
		Total:   total,
	})
}

// ResolveReport handles moderation action on report
// @Summary Обработать репорт
// @Tags Admin Reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID репорта"
// @Param request body ResolveRequest true "Решение по репорту"
// @Success 200 {object} response.Response
// @Failure 400,404,422,500 {object} response.Response
// @Router /admin/reports/{id}/status [patch]
func (h *Handler) ResolveReport(w http.ResponseWriter, r *http.Request) {
	// Parse report ID from URL
	reportID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid report ID")
		return
	}

	// Parse request body
	var req ResolveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	// Validate request
	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	// Get admin ID from context
	adminID := GetAdminID(r.Context())

	// Resolve the report
	if err := h.service.ResolveReport(r.Context(), adminID, reportID, &req); err != nil {
		switch err {
		case ErrReportNotFound:
			response.NotFound(w, "Report not found")
		default:
			response.InternalError(w)
		}
		return
	}

	// Return success response
	response.OK(w, map[string]string{"status": "resolved"})
}
