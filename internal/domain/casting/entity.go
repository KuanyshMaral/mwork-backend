package casting

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Status represents casting status (matches casting_status enum)
type Status string

const (
	StatusDraft  Status = "draft"
	StatusActive Status = "active"
	StatusClosed Status = "closed"
)

// ModerationStatus represents casting moderation status
type ModerationStatus string

const (
	ModerationPending  ModerationStatus = "pending"
	ModerationApproved ModerationStatus = "approved"
	ModerationRejected ModerationStatus = "rejected"
)

// Requirements stored as JSONB in DB
type Requirements struct {
	Gender             string   `json:"gender,omitempty"`
	AgeMin             int      `json:"age_min,omitempty"`
	AgeMax             int      `json:"age_max,omitempty"`
	HeightMin          float64  `json:"height_min,omitempty"`
	HeightMax          float64  `json:"height_max,omitempty"`
	ExperienceRequired bool     `json:"experience_required,omitempty"`
	Languages          []string `json:"languages,omitempty"`
}

// Casting represents a job posting (matches actual castings table)
type Casting struct {
	ID        uuid.UUID `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`

	// Owner (FK to users)
	CreatorID uuid.UUID `db:"creator_id"`

	// Basic info
	Title       string `db:"title"`
	Description string `db:"description"`

	// Location
	City    string         `db:"city"`
	Address sql.NullString `db:"address"`

	// Payment
	PayMin  sql.NullFloat64 `db:"pay_min"`
	PayMax  sql.NullFloat64 `db:"pay_max"`
	PayType string          `db:"pay_type"` // fixed, hourly, negotiable, free

	// Dates
	DateFrom sql.NullTime `db:"date_from"`
	DateTo   sql.NullTime `db:"date_to"`

	// Cover image
	CoverImageURL sql.NullString `db:"cover_image_url"`

	// Requirements (JSONB)
	Requirements json.RawMessage `db:"requirements"`

	// Status and promotion
	Status     Status `db:"status"`
	IsPromoted bool   `db:"is_promoted"`

	// Task 3: Moderation fields
	ModerationStatus ModerationStatus `db:"moderation_status"`

	// Stats
	ViewCount     int `db:"view_count"`
	ResponseCount int `db:"response_count"`

	// Joined data (not in DB, populated by queries)
	CreatorName string `db:"-"`
}

// IsActive returns true if casting is active
func (c *Casting) IsActive() bool {
	return c.Status == StatusActive
}

// IsDraft returns true if casting is draft
func (c *Casting) IsDraft() bool {
	return c.Status == StatusDraft
}

// CanBeEditedBy checks if user can edit this casting
func (c *Casting) CanBeEditedBy(userID uuid.UUID) bool {
	return c.CreatorID == userID
}

// GetPayRange returns formatted pay range
func (c *Casting) GetPayRange() string {
	if !c.PayMin.Valid && !c.PayMax.Valid {
		return "По договоренности"
	}
	if c.PayMin.Valid && c.PayMax.Valid {
		if c.PayMin.Float64 == c.PayMax.Float64 {
			return fmt.Sprintf("%.0f ₸", c.PayMin.Float64)
		}
		return fmt.Sprintf("%.0f - %.0f ₸", c.PayMin.Float64, c.PayMax.Float64)
	}
	if c.PayMin.Valid {
		return fmt.Sprintf("от %.0f ₸", c.PayMin.Float64)
	}
	return fmt.Sprintf("до %.0f ₸", c.PayMax.Float64)
}

// GetRequirements parses requirements JSON
func (c *Casting) GetRequirements() Requirements {
	var req Requirements
	if c.Requirements != nil {
		_ = json.Unmarshal(c.Requirements, &req)
	}
	return req
}

// SetRequirements serializes requirements to JSON
func (c *Casting) SetRequirements(req Requirements) {
	c.Requirements, _ = json.Marshal(req)
}

// IsFree returns true if casting has no payment
func (c *Casting) IsFree() bool {
	return c.PayType == "free" || (!c.PayMin.Valid && !c.PayMax.Valid)
}
