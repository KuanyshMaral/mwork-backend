package promotion

import (
	"time"
)

// CreateRequest for creating new promotion
type CreateRequest struct {
	Title          string   `json:"title" validate:"required,min=5,max=255"`
	Description    string   `json:"description" validate:"max=2000"`
	PhotoURL       string   `json:"photo_url" validate:"omitempty,url"`
	Specialization string   `json:"specialization" validate:"omitempty,max=100"`
	TargetAudience string   `json:"target_audience" validate:"omitempty,oneof=employers agencies all"`
	TargetCities   []string `json:"target_cities" validate:"omitempty,dive,min=2,max=100"`
	BudgetAmount   int64    `json:"budget_amount" validate:"required,gte=1000"`
	DailyBudget    *int64   `json:"daily_budget" validate:"omitempty,gte=500"`
	DurationDays   int      `json:"duration_days" validate:"required,gte=1,lte=90"`
}

// UpdateRequest for updating promotion
type UpdateRequest struct {
	Title          string   `json:"title" validate:"omitempty,min=5,max=255"`
	Description    string   `json:"description" validate:"max=2000"`
	PhotoURL       string   `json:"photo_url" validate:"omitempty,url"`
	Specialization string   `json:"specialization" validate:"omitempty,max=100"`
	TargetAudience string   `json:"target_audience" validate:"omitempty,oneof=employers agencies all"`
	TargetCities   []string `json:"target_cities" validate:"omitempty,dive,min=2,max=100"`
}

// Response for API response
type Response struct {
	ID             string   `json:"id"`
	ProfileID      string   `json:"profile_id"`
	Title          string   `json:"title"`
	Description    string   `json:"description,omitempty"`
	PhotoURL       string   `json:"photo_url,omitempty"`
	Specialization string   `json:"specialization,omitempty"`
	TargetAudience string   `json:"target_audience"`
	TargetCities   []string `json:"target_cities,omitempty"`
	BudgetAmount   int64    `json:"budget_amount"`
	DailyBudget    *int64   `json:"daily_budget,omitempty"`
	DurationDays   int      `json:"duration_days"`
	Status         string   `json:"status"`
	StartsAt       string   `json:"starts_at,omitempty"`
	EndsAt         string   `json:"ends_at,omitempty"`
	Impressions    int      `json:"impressions"`
	Clicks         int      `json:"clicks"`
	Responses      int      `json:"responses"`
	SpentAmount    int64    `json:"spent_amount"`
	CTR            float64  `json:"ctr"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

// ToResponse converts entity to response
func (p *Promotion) ToResponse() *Response {
	resp := &Response{
		ID:             p.ID.String(),
		ProfileID:      p.ProfileID.String(),
		Title:          p.Title,
		TargetAudience: p.TargetAudience,
		TargetCities:   p.TargetCities,
		BudgetAmount:   p.BudgetAmount,
		DurationDays:   p.DurationDays,
		Status:         string(p.Status),
		Impressions:    p.Impressions,
		Clicks:         p.Clicks,
		Responses:      p.Responses,
		SpentAmount:    p.SpentAmount,
		CreatedAt:      p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      p.UpdatedAt.Format(time.RFC3339),
	}

	if p.Description.Valid {
		resp.Description = p.Description.String
	}
	if p.PhotoURL.Valid {
		resp.PhotoURL = p.PhotoURL.String
	}
	if p.Specialization.Valid {
		resp.Specialization = p.Specialization.String
	}
	if p.DailyBudget.Valid {
		v := p.DailyBudget.Int64
		resp.DailyBudget = &v
	}
	if p.StartsAt.Valid {
		resp.StartsAt = p.StartsAt.Time.Format(time.RFC3339)
	}
	if p.EndsAt.Valid {
		resp.EndsAt = p.EndsAt.Time.Format(time.RFC3339)
	}

	// Calculate CTR
	if p.Impressions > 0 {
		resp.CTR = float64(p.Clicks) / float64(p.Impressions) * 100
	}

	return resp
}

// StatsResponse for promotion statistics
type StatsResponse struct {
	TotalImpressions int             `json:"total_impressions"`
	TotalClicks      int             `json:"total_clicks"`
	TotalResponses   int             `json:"total_responses"`
	TotalSpent       int64           `json:"total_spent"`
	CTR              float64         `json:"ctr"`
	ConversionRate   float64         `json:"conversion_rate"`
	DailyStats       []DailyStatItem `json:"daily_stats"`
}

// DailyStatItem for daily breakdown
type DailyStatItem struct {
	Date        string `json:"date"`
	Impressions int    `json:"impressions"`
	Clicks      int    `json:"clicks"`
	Responses   int    `json:"responses"`
	Spent       int64  `json:"spent"`
}
