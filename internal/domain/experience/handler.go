package experience

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mwork/mwork-api/internal/domain/profile"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"
)

// Handler handles work experience HTTP requests
type Handler struct {
	repo        Repository
	profileRepo profile.ModelRepository
}

// NewHandler creates new work experience handler
func NewHandler(repo Repository, profileRepo profile.ModelRepository) *Handler {
	return &Handler{repo: repo, profileRepo: profileRepo}
}

func (h *Handler) resolveProfileByPathID(r *http.Request) (*profile.ModelProfile, error) {
	profileIDStr := chi.URLParam(r, "id")
	if profileIDStr == "" {
		return nil, nil
	}

	id, err := uuid.Parse(profileIDStr)
	if err != nil {
		return nil, err
	}

	p, err := h.profileRepo.GetByID(r.Context(), id)
	if err != nil {
		return nil, err
	}
	if p != nil {
		return p, nil
	}

	// Backward compatibility: some clients send user_id in {id}.
	return h.profileRepo.GetByUserID(r.Context(), id)
}

// Create adds new work experience
// @Summary Добавить опыт работы
// @Tags Experience
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID профиля"
// @Param request body CreateRequest true "Данные опыта"
// @Success 201 {object} response.Response{data=Entity}
// @Failure 400,401,403,404,422,500 {object} response.Response
// @Router /profiles/{id}/experience [post]
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

	p, err := h.resolveProfileByPathID(r)
	if err != nil {
		response.BadRequest(w, "Invalid profile ID")
		return
	}

	if p == nil {
		response.NotFound(w, "Profile not found")
		return
	}
	if p.UserID != userID {
		response.Forbidden(w, "Forbidden")
		return
	}

	// TODO: Verify profile ownership
	// For now, we assume the profile belongs to the user
	// In production, you should verify: profileRepo.GetByID(profileID).UserID == userID

	// Create entity from request
	exp := &Entity{
		ProfileID:   p.ID.String(),
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
// @Summary Список опыта работы профиля
// @Tags Experience
// @Produce json
// @Param id path string true "ID профиля"
// @Success 200 {object} response.Response{data=[]Entity}
// @Failure 400,500 {object} response.Response
// @Router /profiles/{id}/experience [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	p, err := h.resolveProfileByPathID(r)
	if err != nil {
		response.BadRequest(w, "Invalid profile ID")
		return
	}
	if p == nil {
		response.NotFound(w, "Profile not found")
		return
	}

	// Get experiences
	experiences, err := h.repo.ListByProfileID(r.Context(), p.ID.String())
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, experiences)
}

// @Summary Удалить опыт работы
// @Tags Experience
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID профиля"
// @Param expId path string true "ID опыта"
// @Success 204 {string} string "No Content"
// @Failure 400,401,403,404,500 {object} response.Response
// @Router /profiles/{id}/experience/{expId} [delete]
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

	exp, err := h.repo.GetByID(r.Context(), expID)
	if err != nil || exp == nil {
		response.NotFound(w, "Experience not found")
		return
	}

	profileID, err := uuid.Parse(exp.ProfileID)
	if err != nil {
		response.InternalError(w)
		return
	}

	p, err := h.profileRepo.GetByID(r.Context(), profileID)
	if err != nil || p == nil {
		response.NotFound(w, "Profile not found")
		return
	}
	if p.UserID != userID {
		response.Forbidden(w, "Forbidden")
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
