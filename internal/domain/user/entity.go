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
	RoleAdmin    Role = "admin"
)

// Status represents user status (matches user_status enum)
type Status string

const (
	StatusPending   Status = "pending"
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	StatusBanned    Status = "banned"
)

// User represents a user account (matches actual users table)
type User struct {
	ID            uuid.UUID `db:"id"`
	Email         string    `db:"email"`
	PasswordHash  string    `db:"password_hash"`
	Role          Role      `db:"role"`
	EmailVerified bool      `db:"email_verified"`
	IsBanned      bool      `db:"is_banned"`

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

// IsAdmin returns true if user is an admin
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsActive returns true if user is not banned
func (u *User) IsActive() bool {
	return !u.IsBanned
}

// CanCreateCasting returns true if user can create castings
func (u *User) CanCreateCasting() bool {
	return u.IsEmployer() || u.IsAdmin()
}

// CanApplyToCasting returns true if user can apply to castings
func (u *User) CanApplyToCasting() bool {
	return u.IsModel()
}

// ValidRoles returns list of valid roles for registration
func ValidRoles() []Role {
	return []Role{RoleModel, RoleEmployer}
}

// IsValidRole checks if role is valid for registration
func IsValidRole(role string) bool {
	for _, r := range ValidRoles() {
		if string(r) == role {
			return true
		}
	}
	return false
}
