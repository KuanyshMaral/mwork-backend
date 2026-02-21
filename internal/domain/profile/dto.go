package profile

import (
	"time"

	"github.com/google/uuid"
)

// CreateModelProfileRequest defines model profile payload (used for service-level profile creation).
type CreateModelProfileRequest struct {
	Name             string   `json:"name" validate:"required,min=2,max=100"`
	Bio              string   `json:"bio" validate:"max=2000"`
	City             string   `json:"city" validate:"required,min=2,max=100"`
	Age              *int     `json:"age" validate:"omitempty,gte=18,lte=100"`
	Height           *float64 `json:"height" validate:"omitempty,gte=100,lte=250"`
	Weight           *float64 `json:"weight" validate:"omitempty,gte=30,lte=200"`
	Gender           string   `json:"gender" validate:"omitempty,oneof=male female other"`
	HourlyRate       *float64 `json:"hourly_rate" validate:"omitempty,gte=0"`
	Experience       *int     `json:"experience" validate:"omitempty,gte=0,lte=50"`
	Languages        []string `json:"languages"`
	Categories       []string `json:"categories"`
	Skills           []string `json:"skills"`
	BarterAccepted   *bool    `json:"barter_accepted"`
	AcceptRemoteWork *bool    `json:"accept_remote_work"`
	TravelCities     []string `json:"travel_cities"`
	Visibility       string   `json:"visibility" validate:"omitempty,oneof=public link_only hidden"`
}

// UpdateModelProfileRequest for PUT /profiles/models/{id}
type UpdateModelProfileRequest struct {
	Name             string   `json:"name" validate:"omitempty,min=2,max=100"`
	Bio              string   `json:"bio" validate:"max=2000"`
	City             string   `json:"city" validate:"omitempty,min=2,max=100"`
	Age              *int     `json:"age" validate:"omitempty,gte=18,lte=100"`
	Height           *float64 `json:"height" validate:"omitempty,gte=100,lte=250"`
	Weight           *float64 `json:"weight" validate:"omitempty,gte=30,lte=200"`
	Gender           string   `json:"gender" validate:"omitempty,oneof=male female other"`
	HourlyRate       *float64 `json:"hourly_rate" validate:"omitempty,gte=0"`
	Experience       *int     `json:"experience" validate:"omitempty,gte=0,lte=50"`
	Languages        []string `json:"languages"`
	Categories       []string `json:"categories"`
	IsPublic         *bool    `json:"is_public"`
	Visibility       string   `json:"visibility" validate:"omitempty,oneof=public link_only hidden"`
	Skills           []string `json:"skills"`
	BarterAccepted   *bool    `json:"barter_accepted"`
	AcceptRemoteWork *bool    `json:"accept_remote_work"`
	TravelCities     []string `json:"travel_cities"`
	// Physical characteristics
	HairColor string `json:"hair_color" validate:"omitempty,max=50"`
	EyeColor  string `json:"eye_color" validate:"omitempty,max=50"`
	Tattoos   string `json:"tattoos" validate:"omitempty,max=255"`
	// Professional details
	WorkingHours string            `json:"working_hours" validate:"omitempty,max=150"`
	MinBudget    *float64          `json:"min_budget" validate:"omitempty,gte=0"`
	ClothingSize string            `json:"clothing_size" validate:"omitempty,max=20"`
	ShoeSize     string            `json:"shoe_size" validate:"omitempty,max=20"`
	SocialLinks  []SocialLinkEntry `json:"social_links"`
}

// CreateEmployerProfileRequest defines employer profile payload (used for service-level profile creation).
type CreateEmployerProfileRequest struct {
	CompanyName   string `json:"company_name" validate:"required,min=2,max=200"`
	CompanyType   string `json:"company_type" validate:"omitempty,max=100"`
	Description   string `json:"description" validate:"max=2000"`
	City          string `json:"city" validate:"omitempty,min=2,max=100"`
	ContactPerson string `json:"contact_person" validate:"omitempty,max=200"`
	ContactPhone  string `json:"contact_phone" validate:"omitempty,max=20"`
}

// UpdateEmployerProfileRequest for PUT /profiles/employers/{id}
type UpdateEmployerProfileRequest struct {
	CompanyName   string            `json:"company_name" validate:"omitempty,min=2,max=200"`
	CompanyType   string            `json:"company_type" validate:"omitempty,max=100"`
	Description   string            `json:"description" validate:"max=2000"`
	Website       string            `json:"website" validate:"omitempty,url,max=500"`
	City          string            `json:"city" validate:"omitempty,min=2,max=100"`
	ContactPerson string            `json:"contact_person" validate:"omitempty,max=200"`
	ContactPhone  string            `json:"contact_phone" validate:"omitempty,max=20"`
	SocialLinks   []SocialLinkEntry `json:"social_links"`
}

type UpdateAdminProfileRequest struct {
	Name      string `json:"name" validate:"omitempty,max=255"`
	Role      string `json:"role" validate:"omitempty,max=50"`
	AvatarURL string `json:"avatar_url" validate:"omitempty,url,max=2000"`
}

// ModelProfileResponse represents model profile in API response
type ModelProfileResponse struct {
	ID               uuid.UUID `json:"id"`
	UserID           uuid.UUID `json:"user_id"`
	Name             *string   `json:"name,omitempty"`
	Bio              *string   `json:"bio,omitempty"`
	City             *string   `json:"city,omitempty"`
	Country          *string   `json:"country,omitempty"`
	Age              *int      `json:"age,omitempty"`
	Height           *float64  `json:"height,omitempty"`
	Weight           *float64  `json:"weight,omitempty"`
	Gender           *string   `json:"gender,omitempty"`
	HourlyRate       *float64  `json:"hourly_rate,omitempty"`
	Experience       *int      `json:"experience,omitempty"`
	Languages        []string  `json:"languages,omitempty"`
	Categories       []string  `json:"categories,omitempty"`
	Skills           []string  `json:"skills,omitempty"`
	IsPublic         bool      `json:"is_public"`
	Visibility       *string   `json:"visibility,omitempty"`
	ProfileViews     int       `json:"profile_views"`
	Rating           float64   `json:"rating"`
	TotalReviews     int       `json:"total_reviews"`
	CreatedAt        string    `json:"created_at"`
	UpdatedAt        string    `json:"updated_at"`
	BarterAccepted   bool      `json:"barter_accepted"`
	AcceptRemoteWork bool      `json:"accept_remote_work"`
	TravelCities     []string  `json:"travel_cities,omitempty"`
	// Physical characteristics
	HairColor    *string `json:"hair_color,omitempty"`
	EyeColor     *string `json:"eye_color,omitempty"`
	Tattoos      *string `json:"tattoos,omitempty"`
	ClothingSize *string `json:"clothing_size,omitempty"`
	ShoeSize     *string `json:"shoe_size,omitempty"`
	// Professional details
	WorkingHours  *string           `json:"working_hours,omitempty"`
	MinBudget     *float64          `json:"min_budget,omitempty"`
	SocialLinks   []SocialLinkEntry `json:"social_links"`
	AvatarURL     string            `json:"avatar_url,omitempty"`
	CreditBalance int               `json:"credit_balance"`
}

// EmployerProfileResponse represents employer profile in API response
type EmployerProfileResponse struct {
	ID             uuid.UUID         `json:"id"`
	UserID         uuid.UUID         `json:"user_id"`
	CompanyName    string            `json:"company_name"`
	CompanyType    *string           `json:"company_type,omitempty"`
	Description    *string           `json:"description,omitempty"`
	Website        *string           `json:"website,omitempty"`
	City           *string           `json:"city,omitempty"`
	Country        *string           `json:"country,omitempty"`
	ContactPerson  *string           `json:"contact_person,omitempty"`
	ContactPhone   *string           `json:"contact_phone,omitempty"`
	Rating         float64           `json:"rating"`
	TotalReviews   int               `json:"total_reviews"`
	CastingsPosted int               `json:"castings_posted"`
	IsVerified     bool              `json:"is_verified"`
	ProfileViews   int               `json:"profile_views"`
	SocialLinks    []SocialLinkEntry `json:"social_links"`
	CreatedAt      string            `json:"created_at"`
	UpdatedAt      string            `json:"updated_at"`
	CreditBalance  int               `json:"credit_balance"`
}

// CompletenessResponse for profile completeness endpoint
type CompletenessResponse struct {
	Percentage    int      `json:"percentage"`
	MissingFields []string `json:"missing_fields"`
	Tips          []string `json:"tips"`
}

// ModelProfileResponseFromEntity converts entity to response DTO
func ModelProfileResponseFromEntity(p *ModelProfile) *ModelProfileResponse {
	resp := &ModelProfileResponse{
		ID:               p.ID,
		UserID:           p.UserID,
		Languages:        p.GetLanguages(),
		Categories:       p.GetCategories(),
		Skills:           p.GetSkills(),
		IsPublic:         p.IsPublic,
		ProfileViews:     p.ProfileViews,
		Rating:           p.Rating,
		TotalReviews:     p.TotalReviews,
		CreatedAt:        p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        p.UpdatedAt.Format(time.RFC3339),
		BarterAccepted:   p.BarterAccepted,
		AcceptRemoteWork: p.AcceptRemoteWork,
		TravelCities:     p.GetTravelCities(),
		SocialLinks:      p.GetSocialLinks(),
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
	if p.ClothingSize.Valid {
		resp.ClothingSize = &p.ClothingSize.String
	}
	if p.ShoeSize.Valid {
		resp.ShoeSize = &p.ShoeSize.String
	}
	if p.HairColor.Valid {
		resp.HairColor = &p.HairColor.String
	}
	if p.EyeColor.Valid {
		resp.EyeColor = &p.EyeColor.String
	}
	if p.Tattoos.Valid {
		resp.Tattoos = &p.Tattoos.String
	}
	if p.WorkingHours.Valid {
		resp.WorkingHours = &p.WorkingHours.String
	}
	if p.MinBudget.Valid {
		resp.MinBudget = &p.MinBudget.Float64
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
		ProfileViews:   p.ProfileViews,
		SocialLinks:    p.GetSocialLinks(),
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

type AdminProfileResponse struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      *string   `json:"name,omitempty"`
	Role      *string   `json:"role,omitempty"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

func AdminProfileResponseFromEntity(p *AdminProfile) *AdminProfileResponse {
	resp := &AdminProfileResponse{ID: p.ID, UserID: p.UserID, CreatedAt: p.CreatedAt.Format(time.RFC3339), UpdatedAt: p.UpdatedAt.Format(time.RFC3339)}
	if p.Name.Valid {
		resp.Name = &p.Name.String
	}
	if p.Role.Valid {
		resp.Role = &p.Role.String
	}
	if p.AvatarURL.Valid {
		resp.AvatarURL = &p.AvatarURL.String
	}
	return resp
}
