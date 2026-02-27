package promotion

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// CastingPromotion represents an advertising campaign for a casting
type CastingPromotion struct {
	ID         uuid.UUID `db:"id"`
	CastingID  uuid.UUID `db:"casting_id"`
	EmployerID uuid.UUID `db:"employer_id"`

	// Optional overrides
	CustomTitle    sql.NullString `db:"custom_title"`
	CustomPhotoURL sql.NullString `db:"custom_photo_url"`

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

// IsActive returns true if the casting promotion is currently active
func (cp *CastingPromotion) IsActive() bool {
	return cp.Status == StatusActive
}

// CanBeActivated returns true if the casting promotion can be activated
func (cp *CastingPromotion) CanBeActivated() bool {
	return cp.Status == StatusDraft || cp.Status == StatusPaused
}

// CastingPromotionDailyStats represents daily stats for a casting promotion
type CastingPromotionDailyStats struct {
	ID          uuid.UUID `db:"id"`
	PromotionID uuid.UUID `db:"promotion_id"`
	Date        time.Time `db:"date"`
	Impressions int       `db:"impressions"`
	Clicks      int       `db:"clicks"`
	Responses   int       `db:"responses"`
	Spent       int64     `db:"spent"`
}
