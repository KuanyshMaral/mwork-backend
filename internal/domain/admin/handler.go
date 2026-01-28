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

// Handler handles admin HTTP requests
type Handler struct {
	service *Service
	jwtSvc  *JWTService
}

// NewHandler creates admin handler
func NewHandler(service *Service, jwtSvc *JWTService) *Handler {
	return &Handler{
		service: service,
		jwtSvc:  jwtSvc,
	}
}

// --- Authentication ---

// Login handles POST /admin/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	// Get client IP
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.RemoteAddr
	}

	admin, err := h.service.Login(r.Context(), req.Email, req.Password, ip)
	if err != nil {
		switch err {
		case ErrInvalidCredentials:
			response.Unauthorized(w, "Invalid email or password")
		case ErrAdminInactive:
			response.Forbidden(w, "Account is inactive")
		default:
			response.InternalError(w)
		}
		return
	}

	// Generate JWT
	token, err := h.jwtSvc.GenerateToken(admin)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, &LoginResponse{
		AccessToken: token,
		Admin:       AdminResponseFromEntity(admin),
	})
}

// Me handles GET /admin/auth/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	adminID := GetAdminID(r.Context())

	admin, err := h.service.GetAdminByID(r.Context(), adminID)
	if err != nil {
		response.NotFound(w, "Admin not found")
		return
	}

	response.OK(w, AdminResponseFromEntity(admin))
}

// --- Admin Management ---

// ListAdmins handles GET /admin/admins
func (h *Handler) ListAdmins(w http.ResponseWriter, r *http.Request) {
	admins, err := h.service.ListAdmins(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}

	items := make([]*AdminResponse, len(admins))
	for i, a := range admins {
		items[i] = AdminResponseFromEntity(a)
	}

	response.OK(w, items)
}

// CreateAdmin handles POST /admin/admins
func (h *Handler) CreateAdmin(w http.ResponseWriter, r *http.Request) {
	var req CreateAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	actorID := GetAdminID(r.Context())
	admin, err := h.service.CreateAdmin(r.Context(), actorID, &req)
	if err != nil {
		switch err {
		case ErrEmailTaken:
			response.Conflict(w, "Email already in use")
		default:
			response.InternalError(w)
		}
		return
	}

	response.Created(w, AdminResponseFromEntity(admin))
}

// UpdateAdmin handles PATCH /admin/admins/{id}
func (h *Handler) UpdateAdmin(w http.ResponseWriter, r *http.Request) {
	targetID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid admin ID")
		return
	}

	var req UpdateAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	actorID := GetAdminID(r.Context())
	admin, err := h.service.UpdateAdmin(r.Context(), actorID, targetID, &req)
	if err != nil {
		switch err {
		case ErrAdminNotFound:
			response.NotFound(w, "Admin not found")
		case ErrCannotManageRole:
			response.Forbidden(w, "Cannot manage admin with equal or higher role")
		default:
			response.InternalError(w)
		}
		return
	}

	response.OK(w, AdminResponseFromEntity(admin))
}

// --- Feature Flags ---

// ListFeatures handles GET /admin/features
func (h *Handler) ListFeatures(w http.ResponseWriter, r *http.Request) {
	flags, err := h.service.ListFeatureFlags(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, flags)
}

// UpdateFeature handles PATCH /admin/features/{key}
func (h *Handler) UpdateFeature(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	var req FeatureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	adminID := GetAdminID(r.Context())
	if err := h.service.UpdateFeatureFlag(r.Context(), adminID, key, req.Value); err != nil {
		switch err {
		case ErrFeatureFlagNotFound:
			response.NotFound(w, "Feature flag not found")
		default:
			response.InternalError(w)
		}
		return
	}

	response.OK(w, map[string]string{"status": "ok"})
}

// --- Analytics ---

// GetStats returns admin dashboard statistics
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetStats(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, stats)
}

// Dashboard handles GET /admin/analytics/dashboard
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetDashboardStats(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, stats)
}

// --- Audit Logs ---

// AuditLogs handles GET /admin/audit/logs
func (h *Handler) AuditLogs(w http.ResponseWriter, r *http.Request) {
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

	filter := AuditFilter{
		Limit:  limit,
		Offset: offset,
	}

	if action := r.URL.Query().Get("action"); action != "" {
		filter.Action = &action
	}
	if entityType := r.URL.Query().Get("entity_type"); entityType != "" {
		filter.EntityType = &entityType
	}

	logs, total, err := h.service.ListAuditLogs(r.Context(), filter)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]interface{}{
		"items": logs,
		"total": total,
	})
}
