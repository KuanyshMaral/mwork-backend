package profile

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ModelProfile represents a model's profile (matches model_profiles table)
type ModelProfile struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`

	// Basic info
	Name        sql.NullString `db:"name"`
	Bio         sql.NullString `db:"bio"`
	Description sql.NullString `db:"description"`

	// Physical attributes
	Age          sql.NullInt32   `db:"age"`
	Height       sql.NullFloat64 `db:"height"`
	Weight       sql.NullFloat64 `db:"weight"`
	Gender       sql.NullString  `db:"gender"`
	ClothingSize sql.NullString  `db:"clothing_size"`
	ShoeSize     sql.NullString  `db:"shoe_size"`

	// Professional info
	Experience sql.NullInt32   `db:"experience"`
	HourlyRate sql.NullFloat64 `db:"hourly_rate"`
	City       sql.NullString  `db:"city"`
	Country    sql.NullString  `db:"country"`

	// JSON arrays
	Languages  json.RawMessage `db:"languages"`
	Categories json.RawMessage `db:"categories"`
	Skills     json.RawMessage `db:"skills"`

	// Preferences
	BarterAccepted   bool `db:"barter_accepted"`
	AcceptRemoteWork bool `db:"accept_remote_work"`

	// Stats
	ProfileViews int            `db:"profile_views"`
	Rating       float64        `db:"rating"`
	TotalReviews int            `db:"total_reviews"`
	IsPublic     bool           `db:"is_public"`
	Visibility   sql.NullString `db:"visibility"`
}

// EmployerProfile represents an employer's profile (matches employer_profiles table)
type EmployerProfile struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`

	// Company info
	CompanyName string         `db:"company_name"`
	CompanyType sql.NullString `db:"company_type"`
	Description sql.NullString `db:"description"`
	Website     sql.NullString `db:"website"`

	// Contact
	ContactPerson sql.NullString `db:"contact_person"`
	ContactPhone  sql.NullString `db:"contact_phone"`

	// Location
	City    sql.NullString `db:"city"`
	Country sql.NullString `db:"country"`

	// Stats & verification
	Rating         float64      `db:"rating"`
	TotalReviews   int          `db:"total_reviews"`
	CastingsPosted int          `db:"castings_posted"`
	IsVerified     bool         `db:"is_verified"`
	VerifiedAt     sql.NullTime `db:"verified_at"`
}

// GetLanguages parses languages JSON for ModelProfile
func (p *ModelProfile) GetLanguages() []string {
	if p.Languages == nil {
		return []string{}
	}
	var languages []string
	_ = json.Unmarshal(p.Languages, &languages)
	return languages
}

// SetLanguages serializes languages to JSON
func (p *ModelProfile) SetLanguages(languages []string) {
	if languages == nil {
		languages = []string{}
	}
	p.Languages, _ = json.Marshal(languages)
}

// GetCategories parses categories JSON
func (p *ModelProfile) GetCategories() []string {
	if p.Categories == nil {
		return []string{}
	}
	var categories []string
	_ = json.Unmarshal(p.Categories, &categories)
	return categories
}

// SetCategories serializes categories to JSON
func (p *ModelProfile) SetCategories(categories []string) {
	if categories == nil {
		categories = []string{}
	}
	p.Categories, _ = json.Marshal(categories)
}

// GetSkills parses skills JSON
func (p *ModelProfile) GetSkills() []string {
	if p.Skills == nil {
		return []string{}
	}
	var skills []string
	_ = json.Unmarshal(p.Skills, &skills)
	return skills
}

// GetDisplayName returns display name for ModelProfile
func (p *ModelProfile) GetDisplayName() string {
	if p.Name.Valid && p.Name.String != "" {
		return p.Name.String
	}
	return "Model"
}

// GetDisplayName returns display name for EmployerProfile
func (p *EmployerProfile) GetDisplayName() string {
	return p.CompanyName
}

// GetCity returns city for ModelProfile
func (p *ModelProfile) GetCity() string {
	if p.City.Valid {
		return p.City.String
	}
	return ""
}

// GetCity returns city for EmployerProfile
func (p *EmployerProfile) GetCity() string {
	if p.City.Valid {
		return p.City.String
	}
	return ""
}
