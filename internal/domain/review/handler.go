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

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/errorhandler"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

// Handler handles review HTTP requests.
type Handler struct {
	repo      *Repository
	validator *validator.Validate
}

// NewHandler creates a new review handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{
		repo:      repo,
		validator: validator.New(),
	}
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

	targetID, err := uuid.Parse(req.TargetID)
	if err != nil {
		response.BadRequest(w, "Invalid target_id")
		return
	}

	var contextID uuid.NullUUID
	if req.ContextID != "" {
		cid, err := uuid.Parse(req.ContextID)
		if err != nil {
			response.BadRequest(w, "Invalid context_id")
			return
		}
		contextID = uuid.NullUUID{UUID: cid, Valid: true}
	}

	targetType := TargetType(req.TargetType)
	exists, err := h.repo.HasReviewed(r.Context(), userID, targetType, targetID, contextID)
	if err != nil {
		errorhandler.HandleError(r.Context(), w, http.StatusInternalServerError, "REVIEW_CHECK_FAILED", "Failed to check existing review", err)
		return
	}

	if exists {
		response.Conflict(w, "You have already reviewed this entity")
		return
	}

	var contextType sql.NullString
	if req.ContextType != "" {
		contextType = sql.NullString{String: req.ContextType, Valid: true}
	}

	now := time.Now()
	rev := &Review{
		ID:          uuid.New(),
		AuthorID:    userID,
		TargetType:  targetType,
		TargetID:    targetID,
		ContextType: contextType,
		ContextID:   contextID,
		Rating:      req.Rating,
		Comment:     sql.NullString{String: req.Comment, Valid: req.Comment != ""},
		IsPublic:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.repo.Create(r.Context(), rev); err != nil {
		errorhandler.HandleError(r.Context(), w, http.StatusInternalServerError, "REVIEW_CREATION_FAILED", "Failed to create review", err)
		return
	}

	response.Created(w, rev.ToResponse())
}

// ListByTarget handles GET /reviews?target_type=X&target_id=Y
// @Summary Список отзывов
// @Tags Review
// @Produce json
// @Param target_type query string true "Тип сущности (model_profile|employer_profile|casting)"
// @Param target_id query string true "ID сущности"
// @Param page query int false "Страница"
// @Param limit query int false "Лимит (max 50)"
// @Success 200 {object} response.Response{data=[]ReviewResponse}
// @Failure 400,500 {object} response.Response
// @Router /reviews [get]
func (h *Handler) ListByTarget(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rawType := q.Get("target_type")
	rawID := q.Get("target_id")

	if rawType == "" || rawID == "" {
		response.BadRequest(w, "target_type and target_id are required")
		return
	}

	targetID, err := uuid.Parse(rawID)
	if err != nil {
		response.BadRequest(w, "invalid target_id")
		return
	}

	targetType := TargetType(rawType)
	switch targetType {
	case TargetTypeModelProfile, TargetTypeEmployerProfile, TargetTypeCasting:
	default:
		response.BadRequest(w, "invalid target_type")
		return
	}

	page, limit := 1, 10
	if p := q.Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 50 {
			limit = v
		}
	}
	offset := (page - 1) * limit

	reviews, err := h.repo.GetByTarget(r.Context(), targetType, targetID, limit, offset)
	if err != nil {
		errorhandler.HandleError(r.Context(), w, http.StatusInternalServerError, "REVIEW_LIST_FAILED", "Failed to list reviews", err)
		return
	}

	total, err := h.repo.CountByTarget(r.Context(), targetType, targetID)
	if err != nil {
		errorhandler.HandleError(r.Context(), w, http.StatusInternalServerError, "REVIEW_COUNT_FAILED", "Failed to count reviews", err)
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

// GetSummary handles GET /reviews/summary?target_type=X&target_id=Y
// @Summary Сводка рейтинга
// @Tags Review
// @Produce json
// @Param target_type query string true "Тип сущности"
// @Param target_id query string true "ID сущности"
// @Success 200 {object} response.Response{data=TargetRatingSummary}
// @Failure 400,500 {object} response.Response
// @Router /reviews/summary [get]
func (h *Handler) GetSummary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	targetType := TargetType(q.Get("target_type"))
	targetID, err := uuid.Parse(q.Get("target_id"))
	if err != nil {
		response.BadRequest(w, "invalid target_id")
		return
	}

	avg, err := h.repo.GetAverageRating(r.Context(), targetType, targetID)
	if err != nil {
		response.InternalError(w)
		return
	}
	total, err := h.repo.CountByTarget(r.Context(), targetType, targetID)
	if err != nil {
		response.InternalError(w)
		return
	}
	dist, err := h.repo.GetRatingDistribution(r.Context(), targetType, targetID)
	if err != nil {
		response.InternalError(w)
		return
	}
	recent, err := h.repo.GetByTarget(r.Context(), targetType, targetID, 3, 0)
	if err != nil {
		response.InternalError(w)
		return
	}
	recentResp := make([]*ReviewResponse, len(recent))
	for i, rev := range recent {
		recentResp[i] = rev.ToResponse()
	}

	response.OK(w, &TargetRatingSummary{
		AverageRating: avg,
		TotalReviews:  total,
		Distribution:  dist,
		RecentReviews: recentResp,
	})
}

// Delete handles DELETE /reviews/:id
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

	rev, err := h.repo.GetByID(r.Context(), reviewID)
	if err != nil || rev == nil {
		response.NotFound(w, "Review not found")
		return
	}
	if rev.AuthorID != userID {
		response.Forbidden(w, "Can only delete your own reviews")
		return
	}

	if err := h.repo.Delete(r.Context(), reviewID); err != nil {
		response.InternalError(w)
		return
	}
	response.NoContent(w)
}

// Routes returns review routes.
func Routes(h *Handler, authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.ListByTarget)
	r.Get("/summary", h.GetSummary)

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/", h.Create)
		r.Delete("/{id}", h.Delete)
	})

	return r
}
