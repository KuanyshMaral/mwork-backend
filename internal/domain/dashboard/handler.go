package dashboard

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

// Handler handles dashboard HTTP requests
type Handler struct {
	repo *Repository
}

// NewHandler creates new dashboard handler
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// GetModelStats returns aggregated stats for model dashboard
// GET /api/v1/dashboard/model/stats
func (h *Handler) GetModelStats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	stats, err := h.repo.GetModelStats(r.Context(), userID)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, stats)
}

// Routes returns dashboard routes
func Routes(h *Handler, authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Use(authMiddleware)

	r.Get("/model/stats", h.GetModelStats)

	return r
}

// ModelDashboardStats represents aggregated stats for model dashboard
type ModelDashboardStats struct {
	// Profile views
	ProfileViews       int     `json:"profile_views"`
	ProfileViewsChange float64 `json:"profile_views_change_percent"`

	// Responses
	ResponsesUsed      int `json:"responses_used"`
	ResponsesLimit     int `json:"responses_limit"`
	ResponsesThisMonth int `json:"responses_this_month"`

	// Rating
	Rating       float64 `json:"rating"`
	ReviewsCount int     `json:"reviews_count"`

	// Earnings
	TotalEarnings     int64   `json:"total_earnings"`
	EarningsThisMonth int64   `json:"earnings_this_month"`
	EarningsChange    float64 `json:"earnings_change_percent"`

	// Activity
	UnreadMessages   int `json:"unread_messages"`
	PendingResponses int `json:"pending_responses"`
	NewCastingsToday int `json:"new_castings_today"`

	// Subscription
	CurrentPlan   string `json:"current_plan"`
	PlanExpiresAt string `json:"plan_expires_at,omitempty"`
}

// Repository handles dashboard data aggregation
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates new dashboard repository
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// GetModelStats returns aggregated stats for a model
func (r *Repository) GetModelStats(ctx context.Context, userID uuid.UUID) (*ModelDashboardStats, error) {
	stats := &ModelDashboardStats{
		CurrentPlan:    "free",
		ResponsesLimit: 20,
	}

	// Get profile views from profiles table
	_ = r.db.GetContext(ctx, &stats.ProfileViews, `
		SELECT COALESCE(profile_views, 0) 
		FROM profiles WHERE user_id = $1
	`, userID)

	// Get responses count this month
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	_ = r.db.GetContext(ctx, &stats.ResponsesThisMonth, `
		SELECT COUNT(*) FROM responses 
		WHERE user_id = $1 AND created_at >= $2
	`, userID, startOfMonth)

	// Get pending responses
	_ = r.db.GetContext(ctx, &stats.PendingResponses, `
		SELECT COUNT(*) FROM responses 
		WHERE user_id = $1 AND status = 'pending'
	`, userID)

	// Get rating from profile
	_ = r.db.GetContext(ctx, &stats.Rating, `
		SELECT COALESCE(rating, 0) FROM profiles WHERE user_id = $1
	`, userID)

	// Get new castings today
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	_ = r.db.GetContext(ctx, &stats.NewCastingsToday, `
		SELECT COUNT(*) FROM castings 
		WHERE status = 'active' AND created_at >= $1
	`, today)

	stats.ResponsesUsed = stats.ResponsesThisMonth

	// Get earnings from profiles (denormalized)
	_ = r.db.GetContext(ctx, &stats.TotalEarnings, `
		SELECT COALESCE(total_earnings, 0) FROM profiles WHERE user_id = $1
	`, userID)

	return stats, nil
}
