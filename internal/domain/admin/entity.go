package admin

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Role represents admin role
type Role string

const (
	RoleSuperAdmin Role = "super_admin"
	RoleAdmin      Role = "admin"
	RoleModerator  Role = "moderator"
	RoleSupport    Role = "support"
)

// AdminUser represents an admin panel user
type AdminUser struct {
	ID           uuid.UUID      `db:"id" json:"id"`
	Email        string         `db:"email" json:"email"`
	PasswordHash string         `db:"password_hash" json:"-"`
	Role         Role           `db:"role" json:"role"`
	Name         string         `db:"name" json:"name"`
	AvatarURL    sql.NullString `db:"avatar_url" json:"avatar_url,omitempty"`
	IsActive     bool           `db:"is_active" json:"is_active"`
	LastLoginAt  sql.NullTime   `db:"last_login_at" json:"last_login_at,omitempty"`
	LastLoginIP  sql.NullString `db:"last_login_ip" json:"last_login_ip,omitempty"`
	CreatedAt    time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at" json:"updated_at"`
}

// HasPermission checks if admin has a specific permission
func (a *AdminUser) HasPermission(perm Permission) bool {
	permissions, ok := RolePermissions[a.Role]
	if !ok {
		return false
	}
	for _, p := range permissions {
		if p == perm {
			return true
		}
	}
	return false
}

// AuditLog represents an admin action log entry
type AuditLog struct {
	ID         uuid.UUID       `db:"id" json:"id"`
	AdminID    uuid.NullUUID   `db:"admin_id" json:"admin_id,omitempty"`
	AdminEmail string          `db:"admin_email" json:"admin_email"`
	Action     string          `db:"action" json:"action"`
	EntityType string          `db:"entity_type" json:"entity_type"`
	EntityID   uuid.NullUUID   `db:"entity_id" json:"entity_id,omitempty"`
	OldValue   json.RawMessage `db:"old_value" json:"old_value,omitempty"`
	NewValue   json.RawMessage `db:"new_value" json:"new_value,omitempty"`
	Reason     sql.NullString  `db:"reason" json:"reason,omitempty"`
	IPAddress  sql.NullString  `db:"ip_address" json:"ip_address,omitempty"`
	UserAgent  sql.NullString  `db:"user_agent" json:"user_agent,omitempty"`
	CreatedAt  time.Time       `db:"created_at" json:"created_at"`
}

// FeatureFlag represents a runtime feature flag
type FeatureFlag struct {
	Key         string          `db:"key" json:"key"`
	Value       json.RawMessage `db:"value" json:"value"`
	Description sql.NullString  `db:"description" json:"description,omitempty"`
	UpdatedBy   uuid.NullUUID   `db:"updated_by" json:"updated_by,omitempty"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
}

// GetBool returns flag value as bool
func (f *FeatureFlag) GetBool() bool {
	var v bool
	_ = json.Unmarshal(f.Value, &v)
	return v
}

// GetInt returns flag value as int
func (f *FeatureFlag) GetInt() int {
	var v int
	_ = json.Unmarshal(f.Value, &v)
	return v
}

// GetString returns flag value as string
func (f *FeatureFlag) GetString() string {
	var v string
	_ = json.Unmarshal(f.Value, &v)
	return v
}

// ModerationStatus for content
type ModerationStatus string

const (
	ModerationPending  ModerationStatus = "pending"
	ModerationApproved ModerationStatus = "approved"
	ModerationRejected ModerationStatus = "rejected"
)

// Report represents a moderation report
type Report struct {
	ID             uuid.UUID      `db:"id" json:"id"`
	ReporterID     uuid.UUID      `db:"reporter_id" json:"reporter_id"`
	ReportedUserID uuid.UUID      `db:"reported_user_id" json:"reported_user_id"`
	EntityType     string         `db:"entity_type" json:"entity_type"` // user, casting, profile
	EntityID       uuid.UUID      `db:"entity_id" json:"entity_id"`
	Reason         string         `db:"reason" json:"reason"`
	Status         string         `db:"status" json:"status"` // pending, resolved, dismissed
	AdminNotes     sql.NullString `db:"admin_notes" json:"admin_notes,omitempty"`
	ResolvedBy     uuid.NullUUID  `db:"resolved_by" json:"resolved_by,omitempty"`
	ResolvedAt     sql.NullTime   `db:"resolved_at" json:"resolved_at,omitempty"`
	CreatedAt      time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time      `db:"updated_at" json:"updated_at"`
}
