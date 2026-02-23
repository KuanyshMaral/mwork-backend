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
	CoverImageURL string   `json:"cover_image_url" validate:"omitempty,url"`

	// Model requirements
	RequiredGender     string   `json:"required_gender" validate:"omitempty,oneof=male female other"`
	AgeMin             *int     `json:"age_min" validate:"omitempty,gte=0,lte=100"`
	AgeMax             *int     `json:"age_max" validate:"omitempty,gte=0,lte=100"`
	HeightMin          *int     `json:"height_min" validate:"omitempty,gte=0"`
	HeightMax          *int     `json:"height_max" validate:"omitempty,gte=0"`
	WeightMin          *int     `json:"weight_min" validate:"omitempty,gte=0"`
	WeightMax          *int     `json:"weight_max" validate:"omitempty,gte=0"`
	RequiredExperience string   `json:"required_experience" validate:"omitempty,oneof=none beginner medium professional"`
	RequiredLanguages  []string `json:"required_languages" validate:"omitempty"`
	ClothingSizes      []string `json:"clothing_sizes" validate:"omitempty"`
	ShoeSizes          []string `json:"shoe_sizes" validate:"omitempty"`
	RequiredHairColors []string `json:"required_hair_colors" validate:"omitempty"`
	RequiredEyeColors  []string `json:"required_eye_colors" validate:"omitempty"`

	// Work details
	WorkType      string  `json:"work_type" validate:"omitempty,oneof=one_time contract permanent"`
	EventDatetime *string `json:"event_datetime" validate:"omitempty"`
	EventLocation string  `json:"event_location" validate:"omitempty,max=500"`
	DeadlineAt    *string `json:"deadline_at" validate:"omitempty"`
	IsUrgent      bool    `json:"is_urgent"`

	Status string   `json:"status" validate:"omitempty,oneof=draft active"`
	Tags   []string `json:"tags" validate:"omitempty,max=10,dive,max=50"`
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
	CoverImageURL string   `json:"cover_image_url" validate:"omitempty,url"`

	// Model requirements
	RequiredGender     string   `json:"required_gender" validate:"omitempty,oneof=male female other"`
	AgeMin             *int     `json:"age_min" validate:"omitempty,gte=0,lte=100"`
	AgeMax             *int     `json:"age_max" validate:"omitempty,gte=0,lte=100"`
	HeightMin          *int     `json:"height_min" validate:"omitempty,gte=0"`
	HeightMax          *int     `json:"height_max" validate:"omitempty,gte=0"`
	WeightMin          *int     `json:"weight_min" validate:"omitempty,gte=0"`
	WeightMax          *int     `json:"weight_max" validate:"omitempty,gte=0"`
	RequiredExperience string   `json:"required_experience" validate:"omitempty,oneof=none beginner medium professional"`
	RequiredLanguages  []string `json:"required_languages"`
	ClothingSizes      []string `json:"clothing_sizes"`
	ShoeSizes          []string `json:"shoe_sizes"`
	RequiredHairColors []string `json:"required_hair_colors"`
	RequiredEyeColors  []string `json:"required_eye_colors"`

	// Work details
	WorkType      string   `json:"work_type" validate:"omitempty,oneof=one_time contract permanent"`
	EventDatetime *string  `json:"event_datetime"`
	EventLocation string   `json:"event_location" validate:"omitempty,max=500"`
	DeadlineAt    *string  `json:"deadline_at"`
	IsUrgent      *bool    `json:"is_urgent"`
	Tags          []string `json:"tags" validate:"omitempty,max=10,dive,max=50"`
}

// UpdateStatusRequest for PATCH /castings/{id}/status
type UpdateStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=draft active closed"`
}

// RequirementsResponse represents model requirements in API response
type RequirementsResponse struct {
	Gender     string   `json:"gender,omitempty"`
	AgeMin     *int32   `json:"age_min,omitempty"`
	AgeMax     *int32   `json:"age_max,omitempty"`
	HeightMin  *int32   `json:"height_min,omitempty"`
	HeightMax  *int32   `json:"height_max,omitempty"`
	WeightMin  *int32   `json:"weight_min,omitempty"`
	WeightMax  *int32   `json:"weight_max,omitempty"`
	Experience string   `json:"experience,omitempty"`
	Languages  []string `json:"languages,omitempty"`
	Clothing   []string `json:"clothing_sizes,omitempty"`
	Shoes      []string `json:"shoe_sizes,omitempty"`
	HairColors []string `json:"hair_colors,omitempty"`
	EyeColors  []string `json:"eye_colors,omitempty"`
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
	CoverImage    *string `json:"cover_image,omitempty"`

	// Model requirements
	Requirements *RequirementsResponse `json:"requirements,omitempty"`

	// Work details
	WorkType      *string `json:"work_type,omitempty"`
	EventDatetime *string `json:"event_datetime,omitempty"`
	EventLocation *string `json:"event_location,omitempty"`
	DeadlineAt    *string `json:"deadline_at,omitempty"`
	IsUrgent      bool    `json:"is_urgent"`

	Status           string   `json:"status"`
	IsPromoted       bool     `json:"is_promoted"`
	ModerationStatus string   `json:"moderation_status"`
	Tags             []string `json:"tags"`
	ViewCount        int      `json:"view_count"`
	ResponseCount    int      `json:"response_count"`
	Rating           float64  `json:"rating"`
	TotalReviews     int      `json:"total_reviews"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
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
		ModerationStatus: string(c.ModerationStatus),
		Tags:             []string(c.Tags),
		ViewCount:        c.ViewCount,
		ResponseCount:    c.ResponseCount,
		Rating:           c.RatingScore,
		TotalReviews:     c.ReviewsCount,
		IsUrgent:         c.IsUrgent,
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
	if c.CoverURL != "" {
		resp.CoverImageURL = &c.CoverURL
		resp.CoverImage = &c.CoverURL
	} else if c.CoverImageURL.Valid {
		resp.CoverImageURL = &c.CoverImageURL.String
		resp.CoverImage = &c.CoverImageURL.String
	}
	if c.WorkType.Valid {
		resp.WorkType = &c.WorkType.String
	}
	if c.EventDatetime.Valid {
		s := c.EventDatetime.Time.Format(time.RFC3339)
		resp.EventDatetime = &s
	}
	if c.EventLocation.Valid {
		resp.EventLocation = &c.EventLocation.String
	}
	if c.DeadlineAt.Valid {
		s := c.DeadlineAt.Time.Format(time.RFC3339)
		resp.DeadlineAt = &s
	}

	// Build requirements response from dedicated columns
	req := &RequirementsResponse{
		Languages:  []string(c.RequiredLanguages),
		Clothing:   []string(c.ClothingSizes),
		Shoes:      []string(c.ShoeSizes),
		HairColors: []string(c.RequiredHairColors),
		EyeColors:  []string(c.RequiredEyeColors),
	}
	if c.RequiredGender.Valid {
		req.Gender = c.RequiredGender.String
	}
	if c.AgeMin.Valid {
		v := c.AgeMin.Int32
		req.AgeMin = &v
	}
	if c.AgeMax.Valid {
		v := c.AgeMax.Int32
		req.AgeMax = &v
	}
	if c.HeightMin.Valid {
		v := c.HeightMin.Int32
		req.HeightMin = &v
	}
	if c.HeightMax.Valid {
		v := c.HeightMax.Int32
		req.HeightMax = &v
	}
	if c.WeightMin.Valid {
		v := c.WeightMin.Int32
		req.WeightMin = &v
	}
	if c.WeightMax.Valid {
		v := c.WeightMax.Int32
		req.WeightMax = &v
	}
	if c.RequiredExperience.Valid {
		req.Experience = c.RequiredExperience.String
	}

	// Only include requirements if at least one field is set
	if req.Gender != "" || req.AgeMin != nil || req.AgeMax != nil ||
		req.HeightMin != nil || req.HeightMax != nil ||
		req.WeightMin != nil || req.WeightMax != nil ||
		req.Experience != "" || len(req.Languages) > 0 ||
		len(req.Clothing) > 0 || len(req.Shoes) > 0 ||
		len(req.HairColors) > 0 || len(req.EyeColors) > 0 {
		resp.Requirements = req
	}

	return resp
}
