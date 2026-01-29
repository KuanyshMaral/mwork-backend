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
// POST /moderation/block
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
// DELETE /moderation/block
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
// GET /moderation/reports/mine
func (h *Handler) ListMyReports(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	reports, err := h.service.ListMyReports(r.Context(), userID)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, reports)
}

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
// POST /admin/reports/{id}/resolve
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
