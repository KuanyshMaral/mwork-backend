package auth

import (
	"time"

	"github.com/google/uuid"
)

// RegisterRequest for POST /auth/register
type RegisterRequest struct {
	Email           string `json:"email" validate:"required,email,max=255"`
	Password        string `json:"password" validate:"required,min=8,max=128"`
	Role            string `json:"role" validate:"required,oneof=model employer"`
	CompanyName     string `json:"company_name" validate:"required_if=Role employer,omitempty,min=2,max=255"`
	BinIIN          string `json:"bin_iin" validate:"required_if=Role employer,omitempty,len=12"`
	ContactName     string `json:"contact_name" validate:"required_if=Role employer,omitempty,min=2,max=200"`
	ContactPhone    string `json:"contact_phone" validate:"required_if=Role employer,omitempty,min=10,max=20"`
	ContactEmail    string `json:"contact_email" validate:"required_if=Role employer,omitempty,email,max=255"`
	ContactTelegram string `json:"contact_telegram" validate:"omitempty,max=100"`
	ContactWhatsApp string `json:"contact_whatsapp" validate:"omitempty,max=100"`
}

// AgencyRegisterRequest represents agency registration data
type AgencyRegisterRequest struct {
	Email         string `json:"email" validate:"required,email"`
	Password      string `json:"password" validate:"required,min=8"`
	CompanyName   string `json:"company_name" validate:"required,min=2,max=200"`
	Website       string `json:"website" validate:"omitempty,url"`
	ContactPerson string `json:"contact_person" validate:"required,min=2,max=100"`
	Instagram     string `json:"instagram" validate:"omitempty"`
}

// LoginRequest for POST /auth/login
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RefreshRequest for POST /auth/refresh and /auth/logout
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// AuthResponse returned after login/register
type AuthResponse struct {
	User   UserResponse   `json:"user"`
	Tokens TokensResponse `json:"tokens"`
}

// UserResponse represents user in API response
type UserResponse struct {
	ID                 uuid.UUID `json:"id"`
	Email              string    `json:"email"`
	Role               string    `json:"role"`
	EmailVerified      bool      `json:"email_verified"`
	VerificationStatus string    `json:"verification_status"`
	CreatedAt          string    `json:"created_at"`
}

// TokensResponse represents tokens in API response
type TokensResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // seconds until access token expires
}

// NewUserResponse creates UserResponse from user data
func NewUserResponse(id uuid.UUID, email, role string, emailVerified bool, verificationStatus string, createdAt time.Time) UserResponse {
	return UserResponse{
		ID:                 id,
		Email:              email,
		Role:               role,
		EmailVerified:      emailVerified,
		VerificationStatus: verificationStatus,
		CreatedAt:          createdAt.Format(time.RFC3339),
	}
}
