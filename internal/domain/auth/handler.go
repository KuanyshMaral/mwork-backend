package auth

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"
)

// Handler handles auth HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates auth handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register handles POST /auth/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	// First, parse role to determine request type
	var roleCheck struct {
		Role string `json:"role"`
	}

	// Read body into buffer to parse twice
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	// Parse role first
	if err := json.Unmarshal(bodyBytes, &roleCheck); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	// Role-based parsing and registration
	switch roleCheck.Role {
	case "agency":
		var req AgencyRegisterRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			response.BadRequest(w, "Invalid JSON body")
			return
		}

		req.Email = normalizeEmail(req.Email)

		// Validate agency fields
		if errors := validator.Validate(&req); errors != nil {
			response.ValidationError(w, errors)
			return
		}

		// Register agency user
		result, err := h.service.RegisterAgency(r.Context(), &req)
		if err != nil {
			switch err {
			case ErrEmailAlreadyExists:
				response.Conflict(w, "Email already registered")
			default:
				response.InternalError(w)
			}
			return
		}

		response.Created(w, result)

	case "model", "employer":
		var req RegisterRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			response.BadRequest(w, "Invalid JSON body")
			return
		}

		req.Email = normalizeEmail(req.Email)

		// Validate request
		if errors := validator.Validate(&req); errors != nil {
			response.ValidationError(w, errors)
			return
		}

		// Register user
		result, err := h.service.Register(r.Context(), &req)
		if err != nil {
			switch err {
			case ErrEmailAlreadyExists:
				response.Conflict(w, "Email already registered")
			case ErrInvalidRole:
				response.BadRequest(w, "Role must be 'model' or 'employer'")
			default:
				response.InternalError(w)
			}
			return
		}

		response.Created(w, result)

	default:
		response.BadRequest(w, "invalid role")
	}
}

// Login handles POST /auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	// Validate request
	req.Email = normalizeEmail(req.Email)
	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	// Login
	result, err := h.service.Login(r.Context(), &req)
	if err != nil {
		response.Unauthorized(w, "Invalid email or password")
		return
	}

	response.OK(w, result)
}

// Refresh handles POST /auth/refresh
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	// Validate request
	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	// Refresh tokens
	result, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		response.Unauthorized(w, "Invalid or expired refresh token")
		return
	}

	response.OK(w, result)
}

// Logout handles POST /auth/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	// Logout (invalidate refresh token)
	_ = h.service.Logout(r.Context(), req.RefreshToken)

	response.NoContent(w)
}

// Me handles GET /auth/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	// Get current user
	user, err := h.service.GetCurrentUser(r.Context(), userID)
	if err != nil {
		response.NotFound(w, "User not found")
		return
	}

	response.OK(w, user)
}
