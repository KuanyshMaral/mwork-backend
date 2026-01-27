package profile

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

// SocialLinksHandler handles social links HTTP requests
type SocialLinksHandler struct {
	repo      *SocialLinkRepository
	modelRepo ModelRepository
}

// NewSocialLinksHandler creates social links handler
func NewSocialLinksHandler(db *sqlx.DB, modelRepo ModelRepository) *SocialLinksHandler {
	return &SocialLinksHandler{
		repo:      NewSocialLinkRepository(db),
		modelRepo: modelRepo,
	}
}

// List handles GET /profiles/{id}/social-links
func (h *SocialLinksHandler) List(w http.ResponseWriter, r *http.Request) {
	profileID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid profile ID")
		return
	}

	links, err := h.repo.GetByProfileID(r.Context(), profileID)
	if err != nil {
		response.InternalError(w)
		return
	}

	items := make([]SocialLinkResponse, len(links))
	for i, link := range links {
		items[i] = link.ToResponse()
	}

	response.OK(w, items)
}

// Create handles POST /profiles/{id}/social-links
func (h *SocialLinksHandler) Create(w http.ResponseWriter, r *http.Request) {
	profileID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid profile ID")
		return
	}

	// Check ownership via model repo
	userID := middleware.GetUserID(r.Context())
	profile, err := h.modelRepo.GetByID(r.Context(), profileID)
	if err != nil {
		response.NotFound(w, "Profile not found")
		return
	}
	if profile.UserID != userID {
		response.Forbidden(w, "Can only edit your own profile")
		return
	}

	var req SocialLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if !IsValidPlatform(req.Platform) {
		response.BadRequest(w, "Invalid platform. Allowed: instagram, tiktok, facebook, twitter, youtube, telegram, linkedin, vk")
		return
	}

	link := req.ToEntity(profileID)
	if err := h.repo.Create(r.Context(), link); err != nil {
		response.InternalError(w)
		return
	}

	response.Created(w, link.ToResponse())
}

// Delete handles DELETE /profiles/{id}/social-links/{platform}
func (h *SocialLinksHandler) Delete(w http.ResponseWriter, r *http.Request) {
	profileID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid profile ID")
		return
	}

	platform := chi.URLParam(r, "platform")
	if !IsValidPlatform(platform) {
		response.BadRequest(w, "Invalid platform")
		return
	}

	// Check ownership
	userID := middleware.GetUserID(r.Context())
	profile, err := h.modelRepo.GetByID(r.Context(), profileID)
	if err != nil {
		response.NotFound(w, "Profile not found")
		return
	}
	if profile.UserID != userID {
		response.Forbidden(w, "Can only edit your own profile")
		return
	}

	err = h.repo.Delete(r.Context(), profileID, platform)
	if err == sql.ErrNoRows {
		response.NotFound(w, "Social link not found")
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}

	response.NoContent(w)
}

// GetCompleteness handles GET /profiles/{id}/completeness
func (h *SocialLinksHandler) GetCompleteness(w http.ResponseWriter, r *http.Request) {
	profileID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid profile ID")
		return
	}

	profile, err := h.modelRepo.GetByID(r.Context(), profileID)
	if err != nil {
		response.NotFound(w, "Profile not found")
		return
	}

	// Get photo count (simplified - in production would query photo repo)
	photoCount := 0

	completeness := CalculateModelCompleteness(profile, photoCount)

	response.OK(w, completeness)
}

// Routes returns social links routes
func (h *SocialLinksHandler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// Public
	r.Get("/", h.List)

	// Protected
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/", h.Create)
		r.Delete("/{platform}", h.Delete)
	})

	return r
}
