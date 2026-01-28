package experience

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"
)

// Handler handles work experience HTTP requests
type Handler struct {
	repo Repository
}

// NewHandler creates new work experience handler
func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// Create adds new work experience
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	// Validate request
	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	// Get userID from context
	userID := middleware.GetUserID(r.Context())
	if userID.String() == "" {
		response.Unauthorized(w, "unauthorized")
		return
	}

	// TODO: Verify profile ownership
	// For now, we assume the profile belongs to the user
	// In production, you should verify: profileRepo.GetByID(profileID).UserID == userID

	// Create entity from request
	exp := &Entity{
		ProfileID:   req.ProfileID,
		Title:       req.Title,
		Company:     req.Company,
		Role:        req.Role,
		Year:        req.Year,
		Description: req.Description,
	}

	// Create in database
	if err := h.repo.Create(r.Context(), exp); err != nil {
		response.InternalError(w)
		return
	}

	response.Created(w, exp)
}

// List returns all experiences for profile
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	profileID := chi.URLParam(r, "id")
	if profileID == "" {
		response.BadRequest(w, "Missing profile ID")
		return
	}

	// Get experiences
	experiences, err := h.repo.ListByProfileID(r.Context(), profileID)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, experiences)
}

// Delete removes work experience
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	expID := chi.URLParam(r, "expId")
	if expID == "" {
		response.BadRequest(w, "Missing experience ID")
		return
	}

	// Get userID from context
	userID := middleware.GetUserID(r.Context())
	if userID.String() == "" {
		response.Unauthorized(w, "unauthorized")
		return
	}

	// TODO: Verify ownership
	// In production: Get experience by ID, get profile by ID, verify profile.UserID == userID

	// Delete experience
	if err := h.repo.Delete(r.Context(), expID); err != nil {
		response.NotFound(w, "Experience not found")
		return
	}

	response.NoContent(w)
}
