package subscription

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// PlanID represents subscription plan type
type PlanID string

const (
	PlanFree         PlanID = "free"
	PlanFreeModel    PlanID = "free_model"
	PlanFreeEmployer PlanID = "free_employer"
	PlanPro          PlanID = "pro"
	PlanAgency       PlanID = "agency"
)

type Audience string

const (
	AudienceModel    Audience = "model"
	AudienceEmployer Audience = "employer"
)

// Status represents subscription status
type Status string

const (
	StatusActive    Status = "active"
	StatusCancelled Status = "cancelled"
	StatusExpired   Status = "expired"
	StatusPending   Status = "pending"
)

// BillingPeriod represents billing cycle
type BillingPeriod string

const (
	BillingMonthly BillingPeriod = "monthly"
	BillingYearly  BillingPeriod = "yearly"
)

// ConsumablesConfig defines monthly replenishment for Camp 1 (Расходники).
// These are the tangible balances issued each month per plan.
type ConsumablesConfig struct {
	ResponseConnects int `json:"response_connects"` // For Models: how many free response connects are given per month
}

// FeaturesConfig defines Camp 2 limits (Квоты и Фичи) — enforced by count checks, not deductions.
type FeaturesConfig struct {
	MaxPhotos         int  `json:"max_photos"`
	CanChat           bool `json:"can_chat"`
	CanSeeViewers     bool `json:"can_see_viewers"`
	PrioritySearch    bool `json:"priority_search"`
	MaxTeamMembers    int  `json:"max_team_members"`
	MaxActiveCastings int  `json:"max_active_castings"` // Employer: max concurrent active castings
}

// Plan represents a subscription plan
type Plan struct {
	ID           PlanID          `db:"id" json:"id"`
	Name         string          `db:"name" json:"name"`
	Description  string          `db:"description" json:"description"`
	PriceMonthly float64         `db:"price_monthly" json:"price_monthly"`
	PriceYearly  sql.NullFloat64 `db:"price_yearly" json:"price_yearly,omitempty"`
	Audience     Audience        `db:"audience" json:"audience"`
	IsActive     bool            `db:"is_active" json:"is_active"`
	CreatedAt    time.Time       `db:"created_at" json:"created_at"`

	// JSONB columns — raw DB storage
	MonthlyConsumablesRaw []byte `db:"monthly_consumables" json:"-"`
	FeaturesAndQuotasRaw  []byte `db:"features_and_quotas" json:"-"`

	// Parsed structs — populated after scanning
	Consumables ConsumablesConfig `db:"-" json:"consumables"`
	Features    FeaturesConfig    `db:"-" json:"features"`
}

// ParseJSONB parses the raw JSONB fields into typed structs. Must be called after DB scan.
func (p *Plan) ParseJSONB() {
	if len(p.MonthlyConsumablesRaw) > 0 {
		_ = json.Unmarshal(p.MonthlyConsumablesRaw, &p.Consumables)
	}
	if len(p.FeaturesAndQuotasRaw) > 0 {
		_ = json.Unmarshal(p.FeaturesAndQuotasRaw, &p.Features)
	}
}

// Convenience shorthands (replaced old direct fields)
func (p *Plan) MaxPhotos() int         { return p.Features.MaxPhotos }
func (p *Plan) MaxResponsesMonth() int { return p.Consumables.ResponseConnects }
func (p *Plan) CanChat() bool          { return p.Features.CanChat }
func (p *Plan) MaxActiveCastings() int { return p.Features.MaxActiveCastings }

// Subscription represents a user's subscription
type Subscription struct {
	ID            uuid.UUID      `db:"id" json:"id"`
	UserID        uuid.UUID      `db:"user_id" json:"user_id"`
	PlanID        PlanID         `db:"plan_id" json:"plan_id"`
	StartedAt     time.Time      `db:"started_at" json:"started_at"`
	ExpiresAt     sql.NullTime   `db:"expires_at" json:"expires_at,omitempty"`
	Status        Status         `db:"status" json:"status"`
	CancelledAt   sql.NullTime   `db:"cancelled_at" json:"cancelled_at,omitempty"`
	CancelReason  sql.NullString `db:"cancel_reason" json:"cancel_reason,omitempty"`
	BillingPeriod BillingPeriod  `db:"billing_period" json:"billing_period"`
	CreatedAt     time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at" json:"updated_at"`
}

// IsExpired checks if subscription has expired
func (s *Subscription) IsExpired() bool {
	if !s.ExpiresAt.Valid {
		return false // No expiry = never expires (for free)
	}
	return time.Now().After(s.ExpiresAt.Time)
}

// IsActive checks if subscription is active
func (s *Subscription) IsActive() bool {
	return s.Status == StatusActive && !s.IsExpired()
}

// DaysRemaining returns days until expiry
func (s *Subscription) DaysRemaining() int {
	if !s.ExpiresAt.Valid {
		return -1 // Unlimited
	}
	remaining := time.Until(s.ExpiresAt.Time)
	if remaining < 0 {
		return 0
	}
	return int(remaining.Hours() / 24)
}
