package user

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Role represents user role in the system (matches user_role enum)
type Role string

const (
	RoleModel    Role = "model"
	RoleEmployer Role = "employer"
	RoleAgency   Role = "agency"
	RoleAdmin    Role = "admin"
)

// Status represents user status (legacy logical status used in code; users table uses is_banned)
type Status string

const (
	StatusPending   Status = "pending"
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	StatusBanned    Status = "banned"
)

// VerificationStatus represents user's company verification state (matches user_verification_status enum)
type VerificationStatus string

const (
	VerificationNone     VerificationStatus = "none"
	VerificationPending  VerificationStatus = "pending"
	VerificationInReview VerificationStatus = "in_review"
	VerificationVerified VerificationStatus = "verified"
	VerificationRejected VerificationStatus = "rejected"
)

// User represents a user account (matches actual users table)
type User struct {
	ID            uuid.UUID `db:"id"`
	Email         string    `db:"email"`
	PasswordHash  string    `db:"password_hash"`
	Role          Role      `db:"role"`
	EmailVerified bool      `db:"email_verified"`
	IsBanned      bool      `db:"is_banned"`
	CreditBalance int       `db:"credit_balance"`

	// Optional link to organization (for verified employers)
	OrganizationID uuid.NullUUID `db:"organization_id"`

	// Employer/company verification flow
	UserVerificationStatus  VerificationStatus `db:"user_verification_status"`
	VerificationNotes       sql.NullString     `db:"verification_notes"`
	VerificationRejection   sql.NullString     `db:"verification_rejection_reason"`
	VerificationSubmittedAt sql.NullTime       `db:"verification_submitted_at"`
	VerificationReviewedAt  sql.NullTime       `db:"verification_reviewed_at"`
	VerificationReviewedBy  uuid.NullUUID      `db:"verification_reviewed_by"`

	// Verification & Reset tokens
	VerificationToken sql.NullString `db:"verification_token"`
	ResetToken        sql.NullString `db:"reset_token"`
	ResetTokenExp     sql.NullTime   `db:"reset_token_exp"`

	// Login tracking
	LastLoginAt sql.NullTime   `db:"last_login_at"`
	LastLoginIP sql.NullString `db:"last_login_ip"`

	// Two-factor auth
	TwoFactorEnabled bool           `db:"two_factor_enabled"`
	TwoFactorSecret  sql.NullString `db:"two_factor_secret"`

	// Timestamps
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// IsModel returns true if user is a model
func (u *User) IsModel() bool {
	return u.Role == RoleModel
}

// IsEmployer returns true if user is an employer
func (u *User) IsEmployer() bool {
	return u.Role == RoleEmployer
}

// IsAgency returns true if user is an agency
func (u *User) IsAgency() bool {
	return u.Role == RoleAgency
}

// IsAdmin returns true if user is an admin
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsActive returns true if user is not banned
func (u *User) IsActive() bool {
	return !u.IsBanned
}

// IsCompanyVerified returns true if the user's company verification is approved
func (u *User) IsCompanyVerified() bool {
	return u.UserVerificationStatus == VerificationVerified
}

// IsVerificationApproved is an alias used by some domains (casting/chat)
func (u *User) IsVerificationApproved() bool {
	return u.IsCompanyVerified()
}

// CanCreateCasting returns true if user can create castings
func (u *User) CanCreateCasting() bool {
	return u.IsEmployer() || u.IsAgency() || u.IsAdmin()
}

// CanApplyToCasting returns true if user can apply to castings
func (u *User) CanApplyToCasting() bool {
	return u.IsModel()
}

// ValidRoles returns list of valid roles for public registration (model and employer only)
func ValidRoles() []Role {
	return []Role{RoleModel, RoleEmployer}
}

// AllValidRoles returns all valid roles including admin and agency
func AllValidRoles() []Role {
	return []Role{RoleModel, RoleEmployer, RoleAgency, RoleAdmin}
}

// IsValidRole checks if role is valid for public registration
func IsValidRole(role string) bool {
	for _, r := range ValidRoles() {
		if string(r) == role {
			return true
		}
	}
	return false
}
