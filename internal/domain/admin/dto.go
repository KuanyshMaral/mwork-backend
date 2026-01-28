package admin

import (
	"time"

	"github.com/google/uuid"
)

// LoginRequest for POST /admin/auth/login
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// LoginResponse after successful login
type LoginResponse struct {
	AccessToken string         `json:"access_token"`
	Admin       *AdminResponse `json:"admin"`
}

// AdminResponse represents admin in API
type AdminResponse struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	Name        string    `json:"name"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	Permissions []string  `json:"permissions"`
	LastLoginAt *string   `json:"last_login_at,omitempty"`
	CreatedAt   string    `json:"created_at"`
}

// AdminResponseFromEntity converts entity to response
func AdminResponseFromEntity(a *AdminUser) *AdminResponse {
	resp := &AdminResponse{
		ID:        a.ID,
		Email:     a.Email,
		Role:      string(a.Role),
		Name:      a.Name,
		CreatedAt: a.CreatedAt.Format(time.RFC3339),
	}

	if a.AvatarURL.Valid {
		resp.AvatarURL = &a.AvatarURL.String
	}
	if a.LastLoginAt.Valid {
		s := a.LastLoginAt.Time.Format(time.RFC3339)
		resp.LastLoginAt = &s
	}

	// Include permissions
	if perms, ok := RolePermissions[a.Role]; ok {
		resp.Permissions = make([]string, len(perms))
		for i, p := range perms {
			resp.Permissions[i] = string(p)
		}
	}

	return resp
}

// CreateAdminRequest for POST /admin/admins
type CreateAdminRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Role     string `json:"role" validate:"required,oneof=admin moderator support"`
	Name     string `json:"name" validate:"required,min=2,max=100"`
}

// UpdateAdminRequest for PATCH /admin/admins/{id}
type UpdateAdminRequest struct {
	Name     *string `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	Role     *string `json:"role,omitempty" validate:"omitempty,oneof=admin moderator support"`
	IsActive *bool   `json:"is_active,omitempty"`
}

// BanUserRequest for PATCH /admin/users/{id}/ban
type BanUserRequest struct {
	IsBanned bool   `json:"is_banned"`
	Reason   string `json:"reason,omitempty" validate:"max=500"`
}

// VerifyUserRequest for PATCH /admin/users/{id}/verify
type VerifyUserRequest struct {
	IsVerified         *bool `json:"is_verified,omitempty"`
	IsIdentityVerified *bool `json:"is_identity_verified,omitempty"`
}

// ModerateRequest for content moderation
type ModerateRequest struct {
	Status string `json:"status" validate:"required,oneof=approved rejected"`
	Note   string `json:"note,omitempty" validate:"max=500"`
}

// FeatureRequest for PATCH /admin/features/{key}
type FeatureRequest struct {
	Value       interface{} `json:"value" validate:"required"`
	Description *string     `json:"description,omitempty"`
}

// AuditLogResponse represents audit log in API
type AuditLogResponse struct {
	ID         uuid.UUID   `json:"id"`
	AdminEmail string      `json:"admin_email"`
	Action     string      `json:"action"`
	EntityType string      `json:"entity_type"`
	EntityID   *uuid.UUID  `json:"entity_id,omitempty"`
	OldValue   interface{} `json:"old_value,omitempty"`
	NewValue   interface{} `json:"new_value,omitempty"`
	Reason     *string     `json:"reason,omitempty"`
	IPAddress  *string     `json:"ip_address,omitempty"`
	CreatedAt  string      `json:"created_at"`
}

// DashboardStats for /admin/analytics/dashboard
type DashboardStats struct {
	Users      UserStats       `json:"users"`
	Castings   CastingStats    `json:"castings"`
	Responses  ResponseStats   `json:"responses"`
	Revenue    RevenueStats    `json:"revenue"`
	Moderation ModerationStats `json:"moderation"`
}

// StatsResponse represents admin dashboard statistics
type StatsResponse struct {
	TotalUsers          int `json:"total_users"`
	TotalCastings       int `json:"total_castings"`
	ActiveSubscriptions int `json:"active_subscriptions"`
	PendingPayments     int `json:"pending_payments"`
	PendingReports      int `json:"pending_reports"`
}

type UserStats struct {
	Total        int `json:"total"`
	Models       int `json:"models"`
	Employers    int `json:"employers"`
	Active30Days int `json:"active_30_days"`
	NewToday     int `json:"new_today"`
	NewThisWeek  int `json:"new_this_week"`
}

type CastingStats struct {
	Total       int `json:"total"`
	Active      int `json:"active"`
	NewToday    int `json:"new_today"`
	NewThisWeek int `json:"new_this_week"`
}

type ResponseStats struct {
	Total    int `json:"total"`
	Pending  int `json:"pending"`
	Accepted int `json:"accepted"`
	Today    int `json:"today"`
}

type RevenueStats struct {
	TotalKZT     float64 `json:"total_kzt"`
	ThisMonthKZT float64 `json:"this_month_kzt"`
	ProUsers     int     `json:"pro_users"`
	AgencyUsers  int     `json:"agency_users"`
}

type ModerationStats struct {
	PendingProfiles int `json:"pending_profiles"`
	PendingPhotos   int `json:"pending_photos"`
	PendingCastings int `json:"pending_castings"`
	BannedUsers     int `json:"banned_users"`
}

// ListReportsResponse represents paginated reports list
type ListReportsResponse struct {
	Reports []ReportResponse `json:"reports"`
	Total   int              `json:"total"`
}

// ReportResponse represents a report in API
type ReportResponse struct {
	ID             string  `json:"id"`
	ReporterID     string  `json:"reporter_id"`
	ReportedUserID string  `json:"reported_user_id"`
	EntityType     string  `json:"entity_type"` // user, casting, profile
	EntityID       string  `json:"entity_id"`
	Reason         string  `json:"reason"`
	Status         string  `json:"status"` // pending, resolved, dismissed
	AdminNotes     *string `json:"admin_notes,omitempty"`
	CreatedAt      string  `json:"created_at"`
}

// ReportResponseFromEntity converts Report entity to response
func ReportResponseFromEntity(r *Report) ReportResponse {
	resp := ReportResponse{
		ID:             r.ID.String(),
		ReporterID:     r.ReporterID.String(),
		ReportedUserID: r.ReportedUserID.String(),
		EntityType:     r.EntityType,
		EntityID:       r.EntityID.String(),
		Reason:         r.Reason,
		Status:         r.Status,
		CreatedAt:      r.CreatedAt.Format(time.RFC3339),
	}

	if r.AdminNotes.Valid {
		resp.AdminNotes = &r.AdminNotes.String
	}

	return resp
}

// ResolveRequest represents report resolution action
type ResolveRequest struct {
	Action string `json:"action" validate:"required,oneof=warn suspend delete dismiss"`
	Notes  string `json:"notes" validate:"omitempty,max=500"`
}
