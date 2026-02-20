package casting

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
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

// ExperienceLevel represents required experience level
type ExperienceLevel string

const (
	ExperienceNone         ExperienceLevel = "none"
	ExperienceBeginner     ExperienceLevel = "beginner"
	ExperienceMedium       ExperienceLevel = "medium"
	ExperienceProfessional ExperienceLevel = "professional"
)

// WorkType represents the type of work
type WorkType string

const (
	WorkTypeOneTime   WorkType = "one_time"
	WorkTypeContract  WorkType = "contract"
	WorkTypePermanent WorkType = "permanent"
)

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

	// Cover image — two representations during transition:
	// CoverImageURL: legacy string column (kept until migration 000072 drops it).
	// CoverUploadID: new FK to uploads (Phase 4). Use this going forward.
	CoverImageURL sql.NullString `db:"cover_image_url"`
	CoverUploadID uuid.NullUUID  `db:"cover_upload_id"` // Phase 4 FK

	// CoverURL is NOT a DB column — populated by service from upload.GetURL().
	CoverURL string `db:"-" json:"cover_url,omitempty"`

	// Model requirements (dedicated columns from migration 000015)
	RequiredGender     sql.NullString `db:"required_gender"`
	AgeMin             sql.NullInt32  `db:"min_age"`
	AgeMax             sql.NullInt32  `db:"max_age"`
	HeightMin          sql.NullInt32  `db:"min_height"`
	HeightMax          sql.NullInt32  `db:"max_height"`
	WeightMin          sql.NullInt32  `db:"min_weight"`
	WeightMax          sql.NullInt32  `db:"max_weight"`
	RequiredExperience sql.NullString `db:"required_experience"`
	RequiredLanguages  pq.StringArray `db:"required_languages"`
	ClothingSizes      pq.StringArray `db:"clothing_sizes"`
	ShoeSizes          pq.StringArray `db:"shoe_sizes"`

	// Work details (from migration 000015)
	WorkType      sql.NullString `db:"work_type"`
	EventDatetime sql.NullTime   `db:"event_datetime"`
	EventLocation sql.NullString `db:"event_location"`
	DeadlineAt    sql.NullTime   `db:"deadline_at"`
	IsUrgent      bool           `db:"is_urgent"`

	// Status and promotion
	Status     Status `db:"status"`
	IsPromoted bool   `db:"is_promoted"`

	// Tags (user-defined, migration 000064)
	Tags pq.StringArray `db:"tags"`

	// Moderation fields
	ModerationStatus ModerationStatus `db:"moderation_status"`

	// Stats
	ViewCount           int           `db:"view_count"`
	ResponseCount       int           `db:"response_count"`
	RequiredModelsCount sql.NullInt32 `db:"required_models_count"`
	AcceptedModelsCount int           `db:"accepted_models_count"`

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

// IsFree returns true if casting has no payment
func (c *Casting) IsFree() bool {
	return c.PayType == "free" || (!c.PayMin.Valid && !c.PayMax.Valid)
}
