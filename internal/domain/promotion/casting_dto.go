package promotion

import (
	"time"
)

// CreateCastingPromotionRequest for creating a new casting promotion
type CreateCastingPromotionRequest struct {
	CastingID    string `json:"casting_id" validate:"required,uuid"`
	CustomTitle  string `json:"custom_title" validate:"omitempty,max=255"`
	BudgetAmount int64  `json:"budget_amount" validate:"required,gte=1000"`
	DailyBudget  *int64 `json:"daily_budget" validate:"omitempty,gte=500"`
	DurationDays int    `json:"duration_days" validate:"required,gte=1,lte=90"`
}

// CastingPromotionResponse for API response
type CastingPromotionResponse struct {
	ID           string  `json:"id"`
	CastingID    string  `json:"casting_id"`
	EmployerID   string  `json:"employer_id"`
	CustomTitle  string  `json:"custom_title,omitempty"`
	BudgetAmount int64   `json:"budget_amount"`
	DailyBudget  *int64  `json:"daily_budget,omitempty"`
	DurationDays int     `json:"duration_days"`
	Status       string  `json:"status"`
	StartsAt     string  `json:"starts_at,omitempty"`
	EndsAt       string  `json:"ends_at,omitempty"`
	Impressions  int     `json:"impressions"`
	Clicks       int     `json:"clicks"`
	Responses    int     `json:"responses"`
	SpentAmount  int64   `json:"spent_amount"`
	CTR          float64 `json:"ctr"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// ToCastingPromotionResponse converts entity to response DTO
func (cp *CastingPromotion) ToCastingPromotionResponse() *CastingPromotionResponse {
	resp := &CastingPromotionResponse{
		ID:           cp.ID.String(),
		CastingID:    cp.CastingID.String(),
		EmployerID:   cp.EmployerID.String(),
		BudgetAmount: cp.BudgetAmount,
		DurationDays: cp.DurationDays,
		Status:       string(cp.Status),
		Impressions:  cp.Impressions,
		Clicks:       cp.Clicks,
		Responses:    cp.Responses,
		SpentAmount:  cp.SpentAmount,
		CreatedAt:    cp.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    cp.UpdatedAt.Format(time.RFC3339),
	}

	if cp.CustomTitle.Valid {
		resp.CustomTitle = cp.CustomTitle.String
	}
	if cp.DailyBudget.Valid {
		v := cp.DailyBudget.Int64
		resp.DailyBudget = &v
	}
	if cp.StartsAt.Valid {
		resp.StartsAt = cp.StartsAt.Time.Format(time.RFC3339)
	}
	if cp.EndsAt.Valid {
		resp.EndsAt = cp.EndsAt.Time.Format(time.RFC3339)
	}
	if cp.Impressions > 0 {
		resp.CTR = float64(cp.Clicks) / float64(cp.Impressions) * 100
	}

	return resp
}

// CastingPromotionStatsResponse for stats API response
type CastingPromotionStatsResponse struct {
	TotalImpressions int                    `json:"total_impressions"`
	TotalClicks      int                    `json:"total_clicks"`
	TotalResponses   int                    `json:"total_responses"`
	TotalSpent       int64                  `json:"total_spent"`
	CTR              float64                `json:"ctr"`
	ConversionRate   float64                `json:"conversion_rate"`
	DailyStats       []CastingDailyStatItem `json:"daily_stats"`
}

// CastingDailyStatItem for daily breakdown
type CastingDailyStatItem struct {
	Date        string `json:"date"`
	Impressions int    `json:"impressions"`
	Clicks      int    `json:"clicks"`
	Responses   int    `json:"responses"`
	Spent       int64  `json:"spent"`
}
