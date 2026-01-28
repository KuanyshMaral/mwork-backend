package profile

import (
	"time"

	"github.com/google/uuid"
)

// CreateModelProfileRequest for POST /profiles/model
type CreateModelProfileRequest struct {
	Name       string   `json:"name" validate:"required,min=2,max=100"`
	Bio        string   `json:"bio" validate:"max=2000"`
	City       string   `json:"city" validate:"required,min=2,max=100"`
	Age        *int     `json:"age" validate:"omitempty,gte=18,lte=100"`
	Height     *float64 `json:"height" validate:"omitempty,gte=100,lte=250"`
	Weight     *float64 `json:"weight" validate:"omitempty,gte=30,lte=200"`
	Gender     string   `json:"gender" validate:"omitempty,oneof=male female other"`
	HourlyRate *float64 `json:"hourly_rate" validate:"omitempty,gte=0"`
	Experience *int     `json:"experience" validate:"omitempty,gte=0,lte=50"`
	Languages  []string `json:"languages"`
	Categories []string `json:"categories"`
}

// UpdateModelProfileRequest for PUT /profiles/model/{id}
type UpdateModelProfileRequest struct {
	Name       string   `json:"name" validate:"omitempty,min=2,max=100"`
	Bio        string   `json:"bio" validate:"max=2000"`
	City       string   `json:"city" validate:"omitempty,min=2,max=100"`
	Age        *int     `json:"age" validate:"omitempty,gte=18,lte=100"`
	Height     *float64 `json:"height" validate:"omitempty,gte=100,lte=250"`
	Weight     *float64 `json:"weight" validate:"omitempty,gte=30,lte=200"`
	Gender     string   `json:"gender" validate:"omitempty,oneof=male female other"`
	HourlyRate *float64 `json:"hourly_rate" validate:"omitempty,gte=0"`
	Experience *int     `json:"experience" validate:"omitempty,gte=0,lte=50"`
	Languages  []string `json:"languages"`
	Categories []string `json:"categories"`
	IsPublic   *bool    `json:"is_public"`
	Visibility string   `json:"visibility" validate:"omitempty,oneof=public link_only hidden"`
}

// CreateEmployerProfileRequest for POST /profiles/employer
type CreateEmployerProfileRequest struct {
	CompanyName   string `json:"company_name" validate:"required,min=2,max=200"`
	CompanyType   string `json:"company_type" validate:"omitempty,max=100"`
	Description   string `json:"description" validate:"max=2000"`
	City          string `json:"city" validate:"omitempty,min=2,max=100"`
	ContactPerson string `json:"contact_person" validate:"omitempty,max=200"`
	ContactPhone  string `json:"contact_phone" validate:"omitempty,max=20"`
}

// UpdateEmployerProfileRequest for PUT /profiles/employer/{id}
type UpdateEmployerProfileRequest struct {
	CompanyName   string `json:"company_name" validate:"omitempty,min=2,max=200"`
	CompanyType   string `json:"company_type" validate:"omitempty,max=100"`
	Description   string `json:"description" validate:"max=2000"`
	Website       string `json:"website" validate:"omitempty,url,max=500"`
	City          string `json:"city" validate:"omitempty,min=2,max=100"`
	ContactPerson string `json:"contact_person" validate:"omitempty,max=200"`
	ContactPhone  string `json:"contact_phone" validate:"omitempty,max=20"`
}

// ModelProfileResponse represents model profile in API response
type ModelProfileResponse struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	Name         *string   `json:"name,omitempty"`
	Bio          *string   `json:"bio,omitempty"`
	City         *string   `json:"city,omitempty"`
	Country      *string   `json:"country,omitempty"`
	Age          *int      `json:"age,omitempty"`
	Height       *float64  `json:"height,omitempty"`
	Weight       *float64  `json:"weight,omitempty"`
	Gender       *string   `json:"gender,omitempty"`
	HourlyRate   *float64  `json:"hourly_rate,omitempty"`
	Experience   *int      `json:"experience,omitempty"`
	Languages    []string  `json:"languages,omitempty"`
	Categories   []string  `json:"categories,omitempty"`
	Skills       []string  `json:"skills,omitempty"`
	IsPublic     bool      `json:"is_public"`
	Visibility   *string   `json:"visibility,omitempty"`
	ProfileViews int       `json:"profile_views"`
	Rating       float64   `json:"rating"`
	TotalReviews int       `json:"total_reviews"`
	CreatedAt    string    `json:"created_at"`
	UpdatedAt    string    `json:"updated_at"`
}

// EmployerProfileResponse represents employer profile in API response
type EmployerProfileResponse struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	CompanyName    string    `json:"company_name"`
	CompanyType    *string   `json:"company_type,omitempty"`
	Description    *string   `json:"description,omitempty"`
	Website        *string   `json:"website,omitempty"`
	City           *string   `json:"city,omitempty"`
	Country        *string   `json:"country,omitempty"`
	ContactPerson  *string   `json:"contact_person,omitempty"`
	ContactPhone   *string   `json:"contact_phone,omitempty"`
	Rating         float64   `json:"rating"`
	TotalReviews   int       `json:"total_reviews"`
	CastingsPosted int       `json:"castings_posted"`
	IsVerified     bool      `json:"is_verified"`
	CreatedAt      string    `json:"created_at"`
	UpdatedAt      string    `json:"updated_at"`
}

// CompletenessResponse for profile completeness endpoint
type CompletenessResponse struct {
	Percentage    int      `json:"percentage"`
	MissingFields []string `json:"missing_fields"`
	Tips          []string `json:"tips"`
}

// SocialLinkResponse for social links in profile response
type SocialLinkResponse struct {
	Platform   string `json:"platform"`
	URL        string `json:"url"`
	Username   string `json:"username,omitempty"`
	IsVerified bool   `json:"is_verified"`
}

// ModelProfileResponseFromEntity converts entity to response DTO
func ModelProfileResponseFromEntity(p *ModelProfile) *ModelProfileResponse {
	resp := &ModelProfileResponse{
		ID:           p.ID,
		UserID:       p.UserID,
		Languages:    p.GetLanguages(),
		Categories:   p.GetCategories(),
		Skills:       p.GetSkills(),
		IsPublic:     p.IsPublic,
		ProfileViews: p.ProfileViews,
		Rating:       p.Rating,
		TotalReviews: p.TotalReviews,
		CreatedAt:    p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    p.UpdatedAt.Format(time.RFC3339),
	}

	if p.Name.Valid {
		resp.Name = &p.Name.String
	}
	if p.Bio.Valid {
		resp.Bio = &p.Bio.String
	}
	if p.City.Valid {
		resp.City = &p.City.String
	}
	if p.Country.Valid {
		resp.Country = &p.Country.String
	}
	if p.Age.Valid {
		v := int(p.Age.Int32)
		resp.Age = &v
	}
	if p.Height.Valid {
		resp.Height = &p.Height.Float64
	}
	if p.Weight.Valid {
		resp.Weight = &p.Weight.Float64
	}
	if p.Gender.Valid {
		resp.Gender = &p.Gender.String
	}
	if p.HourlyRate.Valid {
		resp.HourlyRate = &p.HourlyRate.Float64
	}
	if p.Experience.Valid {
		v := int(p.Experience.Int32)
		resp.Experience = &v
	}
	if p.Visibility.Valid {
		resp.Visibility = &p.Visibility.String
	}

	return resp
}

// EmployerProfileResponseFromEntity converts entity to response DTO
func EmployerProfileResponseFromEntity(p *EmployerProfile) *EmployerProfileResponse {
	resp := &EmployerProfileResponse{
		ID:             p.ID,
		UserID:         p.UserID,
		CompanyName:    p.CompanyName,
		Rating:         p.Rating,
		TotalReviews:   p.TotalReviews,
		CastingsPosted: p.CastingsPosted,
		IsVerified:     p.IsVerified,
		CreatedAt:      p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      p.UpdatedAt.Format(time.RFC3339),
	}

	if p.CompanyType.Valid {
		resp.CompanyType = &p.CompanyType.String
	}
	if p.Description.Valid {
		resp.Description = &p.Description.String
	}
	if p.Website.Valid {
		resp.Website = &p.Website.String
	}
	if p.City.Valid {
		resp.City = &p.City.String
	}
	if p.Country.Valid {
		resp.Country = &p.Country.String
	}
	if p.ContactPerson.Valid {
		resp.ContactPerson = &p.ContactPerson.String
	}
	if p.ContactPhone.Valid {
		resp.ContactPhone = &p.ContactPhone.String
	}

	return resp
}
