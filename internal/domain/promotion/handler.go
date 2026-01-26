package promotion

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

// Handler handles promotion HTTP requests
type Handler struct {
	repo      *Repository
	validator *validator.Validate
}

// NewHandler creates new promotion handler
func NewHandler(repo *Repository) *Handler {
	return &Handler{
		repo:      repo,
		validator: validator.New(),
	}
}

// List returns user's promotions
// GET /api/v1/promotions
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	promotions, err := h.repo.GetByProfileID(r.Context(), userID)
	if err != nil {
		response.InternalError(w)
		return
	}

	var responses []*Response
	for i := range promotions {
		responses = append(responses, promotions[i].ToResponse())
	}

	response.OK(w, map[string]interface{}{
		"items": responses,
		"total": len(responses),
	})
}

// Get returns a specific promotion
// GET /api/v1/promotions/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid promotion id")
		return
	}

	promo, err := h.repo.GetByID(r.Context(), id)
	if err == ErrPromotionNotFound {
		response.NotFound(w, "promotion not found")
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, promo.ToResponse())
}

// Create creates a new promotion
// POST /api/v1/promotions
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	profileID := middleware.GetUserID(r.Context())
	if profileID == uuid.Nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	// Set defaults
	if req.TargetAudience == "" {
		req.TargetAudience = "employers"
	}

	now := time.Now()
	promo := &Promotion{
		ID:             uuid.New(),
		ProfileID:      profileID,
		Title:          req.Title,
		Description:    sql.NullString{String: req.Description, Valid: req.Description != ""},
		PhotoURL:       sql.NullString{String: req.PhotoURL, Valid: req.PhotoURL != ""},
		Specialization: sql.NullString{String: req.Specialization, Valid: req.Specialization != ""},
		TargetAudience: req.TargetAudience,
		TargetCities:   req.TargetCities,
		BudgetAmount:   req.BudgetAmount,
		DurationDays:   req.DurationDays,
		Status:         StatusDraft,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if req.DailyBudget != nil {
		promo.DailyBudget = sql.NullInt64{Int64: *req.DailyBudget, Valid: true}
	}

	if err := h.repo.Create(r.Context(), promo); err != nil {
		response.InternalError(w)
		return
	}

	response.Created(w, promo.ToResponse())
}

// Activate activates a promotion
// POST /api/v1/promotions/{id}/activate
func (h *Handler) Activate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid promotion id")
		return
	}

	promo, err := h.repo.GetByID(r.Context(), id)
	if err == ErrPromotionNotFound {
		response.NotFound(w, "promotion not found")
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}

	if !promo.CanBeActivated() {
		response.BadRequest(w, "promotion cannot be activated")
		return
	}

	// In real implementation, check payment here
	now := time.Now()
	startsAt := sql.NullTime{Time: now, Valid: true}
	endsAt := sql.NullTime{Time: now.AddDate(0, 0, promo.DurationDays), Valid: true}
	paymentID := uuid.New()

	if err := h.repo.Activate(r.Context(), id, startsAt, endsAt, paymentID); err != nil {
		response.InternalError(w)
		return
	}

	promo, _ = h.repo.GetByID(r.Context(), id)
	response.OK(w, promo.ToResponse())
}

// Pause pauses an active promotion
// POST /api/v1/promotions/{id}/pause
func (h *Handler) Pause(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid promotion id")
		return
	}

	if err := h.repo.UpdateStatus(r.Context(), id, StatusPaused); err != nil {
		if err == ErrPromotionNotFound {
			response.NotFound(w, "promotion not found")
			return
		}
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]string{"status": "paused"})
}

// GetStats returns promotion statistics
// GET /api/v1/promotions/{id}/stats
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid promotion id")
		return
	}

	promo, err := h.repo.GetByID(r.Context(), id)
	if err == ErrPromotionNotFound {
		response.NotFound(w, "promotion not found")
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}

	dailyStats, _ := h.repo.GetDailyStats(r.Context(), id)

	var dailyItems []DailyStatItem
	for _, s := range dailyStats {
		dailyItems = append(dailyItems, DailyStatItem{
			Date:        s.Date.Format("2006-01-02"),
			Impressions: s.Impressions,
			Clicks:      s.Clicks,
			Responses:   s.Responses,
			Spent:       s.Spent,
		})
	}

	stats := &StatsResponse{
		TotalImpressions: promo.Impressions,
		TotalClicks:      promo.Clicks,
		TotalResponses:   promo.Responses,
		TotalSpent:       promo.SpentAmount,
		DailyStats:       dailyItems,
	}

	if promo.Impressions > 0 {
		stats.CTR = float64(promo.Clicks) / float64(promo.Impressions) * 100
	}
	if promo.Clicks > 0 {
		stats.ConversionRate = float64(promo.Responses) / float64(promo.Clicks) * 100
	}

	response.OK(w, stats)
}

// Routes returns promotion routes
func Routes(h *Handler, authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Use(authMiddleware)

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Post("/{id}/activate", h.Activate)
	r.Post("/{id}/pause", h.Pause)
	r.Get("/{id}/stats", h.GetStats)

	return r
}
