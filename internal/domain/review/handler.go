package review

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/mwork/mwork-api/internal/domain/profile"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

// Handler handles review HTTP requests
type Handler struct {
	repo        *Repository
	profileRepo profile.ModelRepository
	validator   *validator.Validate
}

// NewHandler creates new review handler
func NewHandler(repo *Repository, profileRepo profile.ModelRepository) *Handler {
	return &Handler{
		repo:        repo,
		profileRepo: profileRepo,
		validator:   validator.New(),
	}
}

func (h *Handler) resolveProfileID(ctx *http.Request, rawID string) (uuid.UUID, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return uuid.Nil, err
	}

	if p, err := h.profileRepo.GetByID(ctx.Context(), id); err != nil {
		return uuid.Nil, err
	} else if p != nil {
		return p.ID, nil
	}

	if p, err := h.profileRepo.GetByUserID(ctx.Context(), id); err != nil {
		return uuid.Nil, err
	} else if p != nil {
		return p.ID, nil
	}

	return uuid.Nil, nil
}

// Create handles POST /reviews
// @Summary Создать отзыв
// @Tags Review
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateRequest true "Данные отзыва"
// @Success 201 {object} response.Response{data=ReviewResponse}
// @Failure 400,401,409,500 {object} response.Response
// @Router /reviews [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "Unauthorized")
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	profileID, err := h.resolveProfileID(r, req.ProfileID)
	if err != nil {
		response.BadRequest(w, "Invalid profile ID")
		return
	}
	if profileID == uuid.Nil {
		response.NotFound(w, "Profile not found")
		return
	}

	// Check if already reviewed
	var castingID uuid.NullUUID
	if req.CastingID != "" {
		cid, err := uuid.Parse(req.CastingID)
		if err != nil {
			response.BadRequest(w, "Invalid casting ID")
			return
		}
		castingID = uuid.NullUUID{UUID: cid, Valid: true}
	}

	exists, err := h.repo.HasReviewed(r.Context(), profileID, userID, castingID)
	if err != nil {
		response.InternalError(w)
		return
	}
	if exists {
		response.Conflict(w, "You have already reviewed this profile")
		return
	}

	now := time.Now()
	review := &Review{
		ID:         uuid.New(),
		ProfileID:  profileID,
		ReviewerID: userID,
		CastingID:  castingID,
		Rating:     req.Rating,
		Comment:    sql.NullString{String: req.Comment, Valid: req.Comment != ""},
		IsPublic:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := h.repo.Create(r.Context(), review); err != nil {
		response.InternalError(w)
		return
	}

	response.Created(w, review.ToResponse())
}

// ListByProfile handles GET /profiles/{id}/reviews
// @Summary Список отзывов профиля
// @Tags Review
// @Produce json
// @Param id path string true "ID профиля"
// @Param page query int false "Страница"
// @Param limit query int false "Лимит"
// @Success 200 {object} response.Response{data=[]ReviewResponse}
// @Failure 400,500 {object} response.Response
// @Router /profiles/{id}/reviews [get]
func (h *Handler) ListByProfile(w http.ResponseWriter, r *http.Request) {
	profileID, err := h.resolveProfileID(r, chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid profile ID")
		return
	}
	if profileID == uuid.Nil {
		response.NotFound(w, "Profile not found")
		return
	}

	// Pagination
	query := r.URL.Query()
	page := 1
	limit := 10
	if p := query.Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := query.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 50 {
			limit = v
		}
	}

	offset := (page - 1) * limit

	reviews, err := h.repo.GetByProfileID(r.Context(), profileID, limit, offset)
	if err != nil {
		response.InternalError(w)
		return
	}

	total, err := h.repo.CountByProfileID(r.Context(), profileID)
	if err != nil {
		response.InternalError(w)
		return
	}

	items := make([]*ReviewResponse, len(reviews))
	for i, rev := range reviews {
		items[i] = rev.ToResponse()
	}

	response.WithMeta(w, items, response.Meta{
		Total:   total,
		Page:    page,
		Limit:   limit,
		Pages:   (total + limit - 1) / limit,
		HasNext: page*limit < total,
		HasPrev: page > 1,
	})
}

// GetSummary handles GET /profiles/{id}/reviews/summary
// @Summary Сводка рейтинга профиля
// @Tags Review
// @Produce json
// @Param id path string true "ID профиля"
// @Success 200 {object} response.Response{data=ProfileRatingSummary}
// @Failure 400,500 {object} response.Response
// @Router /profiles/{id}/reviews/summary [get]
func (h *Handler) GetSummary(w http.ResponseWriter, r *http.Request) {
	profileID, err := h.resolveProfileID(r, chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid profile ID")
		return
	}
	if profileID == uuid.Nil {
		response.NotFound(w, "Profile not found")
		return
	}

	avg, err := h.repo.GetAverageRating(r.Context(), profileID)
	if err != nil {
		response.InternalError(w)
		return
	}
	total, err := h.repo.CountByProfileID(r.Context(), profileID)
	if err != nil {
		response.InternalError(w)
		return
	}
	dist, err := h.repo.GetRatingDistribution(r.Context(), profileID)
	if err != nil {
		response.InternalError(w)
		return
	}

	// Get recent reviews
	recent, err := h.repo.GetByProfileID(r.Context(), profileID, 3, 0)
	if err != nil {
		response.InternalError(w)
		return
	}
	recentResp := make([]*ReviewResponse, len(recent))
	for i, rev := range recent {
		recentResp[i] = rev.ToResponse()
	}

	summary := &ProfileRatingSummary{
		AverageRating: avg,
		TotalReviews:  total,
		Distribution:  dist,
		RecentReviews: recentResp,
	}

	response.OK(w, summary)
}

// Delete handles DELETE /reviews/{id}
// @Summary Удалить отзыв
// @Tags Review
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID отзыва"
// @Success 204 {string} string "No Content"
// @Failure 400,401,403,404,500 {object} response.Response
// @Router /reviews/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	reviewID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid review ID")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "Unauthorized")
		return
	}

	review, err := h.repo.GetByID(r.Context(), reviewID)
	if err != nil || review == nil {
		response.NotFound(w, "Review not found")
		return
	}

	// Only reviewer can delete their own review
	if review.ReviewerID != userID {
		response.Forbidden(w, "Can only delete your own reviews")
		return
	}

	if err := h.repo.Delete(r.Context(), reviewID); err != nil {
		response.InternalError(w)
		return
	}

	response.NoContent(w)
}

// Routes returns review routes
func Routes(h *Handler, authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/", h.Create)
		r.Delete("/{id}", h.Delete)
	})

	return r
}
