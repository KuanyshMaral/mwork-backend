package promotion

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Status represents promotion status
type Status string

const (
	StatusDraft          Status = "draft"
	StatusPendingPayment Status = "pending_payment"
	StatusActive         Status = "active"
	StatusPaused         Status = "paused"
	StatusCompleted      Status = "completed"
	StatusCancelled      Status = "cancelled"
)

// Promotion represents a profile advertising campaign
type Promotion struct {
	ID        uuid.UUID `db:"id"`
	ProfileID uuid.UUID `db:"profile_id"`

	// Content
	Title          string         `db:"title"`
	Description    sql.NullString `db:"description"`
	PhotoURL       sql.NullString `db:"photo_url"`
	Specialization sql.NullString `db:"specialization"`

	// Targeting
	TargetAudience string   `db:"target_audience"`
	TargetCities   []string `db:"target_cities"`

	// Budget & Duration
	BudgetAmount int64         `db:"budget_amount"`
	DailyBudget  sql.NullInt64 `db:"daily_budget"`
	DurationDays int           `db:"duration_days"`

	// Schedule
	Status   Status       `db:"status"`
	StartsAt sql.NullTime `db:"starts_at"`
	EndsAt   sql.NullTime `db:"ends_at"`

	// Analytics
	Impressions int   `db:"impressions"`
	Clicks      int   `db:"clicks"`
	Responses   int   `db:"responses"`
	SpentAmount int64 `db:"spent_amount"`

	// Payment
	PaymentID *uuid.UUID `db:"payment_id"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// IsActive returns true if promotion is currently active
func (p *Promotion) IsActive() bool {
	return p.Status == StatusActive
}

// CanBeActivated returns true if promotion can be activated
func (p *Promotion) CanBeActivated() bool {
	return p.Status == StatusDraft || p.Status == StatusPaused
}

// DailyStats represents daily promotion statistics
type DailyStats struct {
	ID          uuid.UUID `db:"id"`
	PromotionID uuid.UUID `db:"promotion_id"`
	Date        time.Time `db:"date"`
	Impressions int       `db:"impressions"`
	Clicks      int       `db:"clicks"`
	Responses   int       `db:"responses"`
	Spent       int64     `db:"spent"`
}
