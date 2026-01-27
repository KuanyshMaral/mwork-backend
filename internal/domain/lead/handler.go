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
func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetStats(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, stats)
}

// Reject handles POST /admin/leads/{id}/reject
func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid lead ID")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
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
