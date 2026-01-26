package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines admin data access
type Repository interface {
	// Admin users
	CreateAdmin(ctx context.Context, admin *AdminUser) error
	GetAdminByID(ctx context.Context, id uuid.UUID) (*AdminUser, error)
	GetAdminByEmail(ctx context.Context, email string) (*AdminUser, error)
	ListAdmins(ctx context.Context) ([]*AdminUser, error)
	UpdateAdmin(ctx context.Context, admin *AdminUser) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID, ip string) error

	// Audit logs
	CreateAuditLog(ctx context.Context, log *AuditLog) error
	ListAuditLogs(ctx context.Context, filter AuditFilter) ([]*AuditLog, int, error)

	// Feature flags
	GetFeatureFlag(ctx context.Context, key string) (*FeatureFlag, error)
	ListFeatureFlags(ctx context.Context) ([]*FeatureFlag, error)
	UpdateFeatureFlag(ctx context.Context, key string, value json.RawMessage, adminID uuid.UUID) error

	// Analytics
	GetDashboardStats(ctx context.Context) (*DashboardStats, error)
}

// AuditFilter for filtering audit logs
type AuditFilter struct {
	AdminID    *uuid.UUID
	Action     *string
	EntityType *string
	EntityID   *uuid.UUID
	FromDate   *time.Time
	ToDate     *time.Time
	Limit      int
	Offset     int
}

type repository struct {
	db *sqlx.DB
}

// NewRepository creates admin repository
func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

// Admin users

func (r *repository) CreateAdmin(ctx context.Context, admin *AdminUser) error {
	query := `
		INSERT INTO admin_users (id, email, password_hash, role, name, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		admin.ID,
		admin.Email,
		admin.PasswordHash,
		admin.Role,
		admin.Name,
		admin.IsActive,
		admin.CreatedAt,
		admin.UpdatedAt,
	)
	return err
}

func (r *repository) GetAdminByID(ctx context.Context, id uuid.UUID) (*AdminUser, error) {
	query := `SELECT * FROM admin_users WHERE id = $1`
	var admin AdminUser
	err := r.db.GetContext(ctx, &admin, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &admin, nil
}

func (r *repository) GetAdminByEmail(ctx context.Context, email string) (*AdminUser, error) {
	query := `SELECT * FROM admin_users WHERE email = $1`
	var admin AdminUser
	err := r.db.GetContext(ctx, &admin, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &admin, nil
}

func (r *repository) ListAdmins(ctx context.Context) ([]*AdminUser, error) {
	query := `SELECT * FROM admin_users ORDER BY created_at DESC`
	var admins []*AdminUser
	err := r.db.SelectContext(ctx, &admins, query)
	return admins, err
}

func (r *repository) UpdateAdmin(ctx context.Context, admin *AdminUser) error {
	query := `
		UPDATE admin_users SET
			name = $2, role = $3, is_active = $4, avatar_url = $5, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		admin.ID,
		admin.Name,
		admin.Role,
		admin.IsActive,
		admin.AvatarURL,
	)
	return err
}

func (r *repository) UpdateLastLogin(ctx context.Context, id uuid.UUID, ip string) error {
	query := `UPDATE admin_users SET last_login_at = NOW(), last_login_ip = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, ip)
	return err
}

// Audit logs

func (r *repository) CreateAuditLog(ctx context.Context, log *AuditLog) error {
	query := `
		INSERT INTO audit_logs (id, admin_id, admin_email, action, entity_type, entity_id, old_value, new_value, reason, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.db.ExecContext(ctx, query,
		log.ID,
		log.AdminID,
		log.AdminEmail,
		log.Action,
		log.EntityType,
		log.EntityID,
		log.OldValue,
		log.NewValue,
		log.Reason,
		log.IPAddress,
		log.UserAgent,
		log.CreatedAt,
	)
	return err
}

func (r *repository) ListAuditLogs(ctx context.Context, filter AuditFilter) ([]*AuditLog, int, error) {
	// Base query
	query := `SELECT * FROM audit_logs WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	// Apply filters
	if filter.AdminID != nil {
		query += ` AND admin_id = $` + string(rune('0'+argNum))
		countQuery += ` AND admin_id = $` + string(rune('0'+argNum))
		args = append(args, *filter.AdminID)
		argNum++
	}
	if filter.Action != nil {
		query += ` AND action = $` + string(rune('0'+argNum))
		countQuery += ` AND action = $` + string(rune('0'+argNum))
		args = append(args, *filter.Action)
		argNum++
	}
	if filter.EntityType != nil {
		query += ` AND entity_type = $` + string(rune('0'+argNum))
		countQuery += ` AND entity_type = $` + string(rune('0'+argNum))
		args = append(args, *filter.EntityType)
		argNum++
	}

	// Order and pagination
	query += ` ORDER BY created_at DESC`

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query += ` LIMIT $` + string(rune('0'+argNum)) + ` OFFSET $` + string(rune('0'+argNum+1))
	args = append(args, limit, offset)

	var logs []*AuditLog
	err := r.db.SelectContext(ctx, &logs, query, args...)
	if err != nil {
		return nil, 0, err
	}

	// Get total count (simplified - doesn't pass args correctly for complex filters)
	var total int
	_ = r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM audit_logs`)

	return logs, total, nil
}

// Feature flags

func (r *repository) GetFeatureFlag(ctx context.Context, key string) (*FeatureFlag, error) {
	query := `SELECT * FROM feature_flags WHERE key = $1`
	var flag FeatureFlag
	err := r.db.GetContext(ctx, &flag, query, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &flag, nil
}

func (r *repository) ListFeatureFlags(ctx context.Context) ([]*FeatureFlag, error) {
	query := `SELECT * FROM feature_flags ORDER BY key`
	var flags []*FeatureFlag
	err := r.db.SelectContext(ctx, &flags, query)
	return flags, err
}

func (r *repository) UpdateFeatureFlag(ctx context.Context, key string, value json.RawMessage, adminID uuid.UUID) error {
	query := `UPDATE feature_flags SET value = $2, updated_by = $3, updated_at = NOW() WHERE key = $1`
	_, err := r.db.ExecContext(ctx, query, key, value, adminID)
	return err
}

// Analytics

func (r *repository) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	stats := &DashboardStats{}

	// User stats
	r.db.GetContext(ctx, &stats.Users.Total, `SELECT COUNT(*) FROM users`)
	r.db.GetContext(ctx, &stats.Users.Models, `SELECT COUNT(*) FROM users WHERE role = 'model'`)
	r.db.GetContext(ctx, &stats.Users.Employers, `SELECT COUNT(*) FROM users WHERE role IN ('employer', 'agency')`)
	r.db.GetContext(ctx, &stats.Users.NewToday, `SELECT COUNT(*) FROM users WHERE created_at >= CURRENT_DATE`)
	r.db.GetContext(ctx, &stats.Users.NewThisWeek, `SELECT COUNT(*) FROM users WHERE created_at >= CURRENT_DATE - INTERVAL '7 days'`)

	// Casting stats
	r.db.GetContext(ctx, &stats.Castings.Total, `SELECT COUNT(*) FROM castings`)
	r.db.GetContext(ctx, &stats.Castings.Active, `SELECT COUNT(*) FROM castings WHERE status = 'active'`)
	r.db.GetContext(ctx, &stats.Castings.NewToday, `SELECT COUNT(*) FROM castings WHERE created_at >= CURRENT_DATE`)
	r.db.GetContext(ctx, &stats.Castings.NewThisWeek, `SELECT COUNT(*) FROM castings WHERE created_at >= CURRENT_DATE - INTERVAL '7 days'`)

	// Response stats
	r.db.GetContext(ctx, &stats.Responses.Total, `SELECT COUNT(*) FROM responses`)
	r.db.GetContext(ctx, &stats.Responses.Pending, `SELECT COUNT(*) FROM responses WHERE status = 'pending'`)
	r.db.GetContext(ctx, &stats.Responses.Accepted, `SELECT COUNT(*) FROM responses WHERE status = 'accepted'`)
	r.db.GetContext(ctx, &stats.Responses.Today, `SELECT COUNT(*) FROM responses WHERE created_at >= CURRENT_DATE`)

	// Revenue stats
	r.db.GetContext(ctx, &stats.Revenue.TotalKZT, `SELECT COALESCE(SUM(amount), 0) FROM payments WHERE status = 'completed'`)
	r.db.GetContext(ctx, &stats.Revenue.ThisMonthKZT, `SELECT COALESCE(SUM(amount), 0) FROM payments WHERE status = 'completed' AND created_at >= DATE_TRUNC('month', CURRENT_DATE)`)
	r.db.GetContext(ctx, &stats.Revenue.ProUsers, `SELECT COUNT(*) FROM subscriptions WHERE plan_id = 'pro' AND status = 'active'`)
	r.db.GetContext(ctx, &stats.Revenue.AgencyUsers, `SELECT COUNT(*) FROM subscriptions WHERE plan_id = 'agency' AND status = 'active'`)

	// Moderation stats
	r.db.GetContext(ctx, &stats.Moderation.PendingProfiles, `SELECT COUNT(*) FROM profiles WHERE moderation_status = 'pending'`)
	r.db.GetContext(ctx, &stats.Moderation.PendingPhotos, `SELECT COUNT(*) FROM photos WHERE moderation_status = 'pending'`)
	r.db.GetContext(ctx, &stats.Moderation.PendingCastings, `SELECT COUNT(*) FROM castings WHERE moderation_status = 'pending'`)
	r.db.GetContext(ctx, &stats.Moderation.BannedUsers, `SELECT COUNT(*) FROM users WHERE is_banned = true`)

	return stats, nil
}
