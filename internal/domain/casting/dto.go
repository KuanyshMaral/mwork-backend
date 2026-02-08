package casting

import (
	"time"

	"github.com/google/uuid"
)

// CreateCastingRequest for POST /castings
type CreateCastingRequest struct {
	Title         string   `json:"title" validate:"required,min=5,max=200"`
	Description   string   `json:"description" validate:"required,min=20,max=5000"`
	City          string   `json:"city" validate:"required,min=2,max=100"`
	Address       string   `json:"address" validate:"omitempty,max=500"`
	PayMin        *float64 `json:"pay_min" validate:"omitempty,gte=0"`
	PayMax        *float64 `json:"pay_max" validate:"omitempty,gte=0"`
	PayType       string   `json:"pay_type" validate:"omitempty,oneof=fixed hourly negotiable free"`
	DateFrom      *string  `json:"date_from" validate:"omitempty"`
	DateTo        *string  `json:"date_to" validate:"omitempty"`
	CoverImageURL string   `json:"cover_image_url" validate:"omitempty,url,startswith=https://"`

	// Requirements (stored as JSONB)
	Requirements *RequirementsRequest `json:"requirements"`

	Status string `json:"status" validate:"omitempty,oneof=draft active"`
}

// RequirementsRequest for nested requirements in create/update
type RequirementsRequest struct {
	Gender             string   `json:"gender,omitempty"`
	AgeMin             int      `json:"age_min,omitempty"`
	AgeMax             int      `json:"age_max,omitempty"`
	HeightMin          float64  `json:"height_min,omitempty"`
	HeightMax          float64  `json:"height_max,omitempty"`
	ExperienceRequired bool     `json:"experience_required,omitempty"`
	Languages          []string `json:"languages,omitempty"`
}

// UpdateCastingRequest for PUT /castings/{id}
type UpdateCastingRequest struct {
	Title         string   `json:"title" validate:"omitempty,min=5,max=200"`
	Description   string   `json:"description" validate:"omitempty,min=20,max=5000"`
	City          string   `json:"city" validate:"omitempty,min=2,max=100"`
	Address       string   `json:"address" validate:"omitempty,max=500"`
	PayMin        *float64 `json:"pay_min" validate:"omitempty,gte=0"`
	PayMax        *float64 `json:"pay_max" validate:"omitempty,gte=0"`
	PayType       string   `json:"pay_type" validate:"omitempty,oneof=fixed hourly negotiable free"`
	DateFrom      *string  `json:"date_from"`
	DateTo        *string  `json:"date_to"`
	CoverImageURL string   `json:"cover_image_url" validate:"omitempty,url,startswith=https://"`

	Requirements *RequirementsRequest `json:"requirements"`
}

// UpdateStatusRequest for PATCH /castings/{id}/status
type UpdateStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=draft active closed"`
}

// CastingResponse represents casting in API response
type CastingResponse struct {
	ID          uuid.UUID `json:"id"`
	CreatorID   uuid.UUID `json:"creator_id"`
	CreatorName string    `json:"creator_name,omitempty"`

	Title       string  `json:"title"`
	Description string  `json:"description"`
	City        string  `json:"city"`
	Address     *string `json:"address,omitempty"`

	PayMin   *float64 `json:"pay_min,omitempty"`
	PayMax   *float64 `json:"pay_max,omitempty"`
	PayType  string   `json:"pay_type"`
	PayRange string   `json:"pay_range"`

	DateFrom *string `json:"date_from,omitempty"`
	DateTo   *string `json:"date_to,omitempty"`

	CoverImageURL *string `json:"cover_image_url,omitempty"`

	// Requirements (from JSONB)
	Requirements *Requirements `json:"requirements,omitempty"`

	Status           string `json:"status"`
	IsPromoted       bool   `json:"is_promoted"`
	ModerationStatus string `json:"moderation_status"` // Task 3: Added moderation status
	ViewCount        int    `json:"view_count"`
	ResponseCount    int    `json:"response_count"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// CastingResponseFromEntity converts entity to response DTO
func CastingResponseFromEntity(c *Casting) *CastingResponse {
	resp := &CastingResponse{
		ID:               c.ID,
		CreatorID:        c.CreatorID,
		CreatorName:      c.CreatorName,
		Title:            c.Title,
		Description:      c.Description,
		City:             c.City,
		PayType:          c.PayType,
		PayRange:         c.GetPayRange(),
		Status:           string(c.Status),
		IsPromoted:       c.IsPromoted,
		ModerationStatus: string(c.ModerationStatus), // Task 3: Include moderation status
		ViewCount:        c.ViewCount,
		ResponseCount:    c.ResponseCount,
		CreatedAt:        c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        c.UpdatedAt.Format(time.RFC3339),
	}

	if c.Address.Valid {
		resp.Address = &c.Address.String
	}
	if c.PayMin.Valid {
		resp.PayMin = &c.PayMin.Float64
	}
	if c.PayMax.Valid {
		resp.PayMax = &c.PayMax.Float64
	}
	if c.DateFrom.Valid {
		s := c.DateFrom.Time.Format(time.RFC3339)
		resp.DateFrom = &s
	}
	if c.DateTo.Valid {
		s := c.DateTo.Time.Format(time.RFC3339)
		resp.DateTo = &s
	}
	if c.CoverImageURL.Valid {
		resp.CoverImageURL = &c.CoverImageURL.String
	}

	// Parse requirements from JSONB
	if c.Requirements != nil && len(c.Requirements) > 0 {
		req := c.GetRequirements()
		resp.Requirements = &req
	}

	return resp
}
