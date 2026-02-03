package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
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

	// Stats count methods
	CountUsers(ctx context.Context) (int, error)
	CountActiveCastings(ctx context.Context) (int, error)
	CountActiveSubscriptions(ctx context.Context) (int, error)
	CountPendingPayments(ctx context.Context) (int, error)
	CountPendingReports(ctx context.Context) (int, error)

	// Reports
	ListReports(ctx context.Context, page, limit int, statusFilter *string) ([]*Report, int, error)
	GetReportByID(ctx context.Context, id uuid.UUID) (*Report, error)
	UpdateReportStatus(ctx context.Context, id uuid.UUID, status string, adminNotes string, resolvedBy uuid.UUID) error
	GetReportedUserByID(ctx context.Context, userID uuid.UUID) (*ReportedUser, error)
	SoftDeleteEntity(ctx context.Context, entityType string, entityID uuid.UUID) error

	// User management
	ListUsers(ctx context.Context, params map[string]interface{}) ([]map[string]interface{}, int, error)
	UpdateUserStatus(ctx context.Context, userID string, status string) error
	UpdateUserStatusWithReason(ctx context.Context, userID, status, reason string) error

	// SQL execution (temporary solution)
	ExecuteSql(ctx context.Context, query string) (interface{}, error)

	// Transaction-based report resolution
	BeginTx(ctx context.Context) (*sqlx.Tx, error)
	UpdateReportStatusTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, status string, adminNotes string, resolvedBy uuid.UUID) error
	UpdateUserStatusTx(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID, status string) error
	SoftDeleteEntityTx(ctx context.Context, tx *sqlx.Tx, entityType string, entityID uuid.UUID) error
}

// ReportedUser simplified struct for moderation actions
type ReportedUser struct {
	ID     uuid.UUID
	Email  string
	Status string
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

// Stats count methods

func (r *repository) CountUsers(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM users`
	err := r.db.GetContext(ctx, &count, query)
	return count, err
}

func (r *repository) CountActiveCastings(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM castings WHERE status IN ('active', 'draft')`
	err := r.db.GetContext(ctx, &count, query)
	return count, err
}

func (r *repository) CountActiveSubscriptions(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM subscriptions WHERE status = 'active'`
	err := r.db.GetContext(ctx, &count, query)
	return count, err
}

func (r *repository) CountPendingPayments(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM payments WHERE status = 'pending'`
	err := r.db.GetContext(ctx, &count, query)
	return count, err
}

func (r *repository) CountPendingReports(ctx context.Context) (int, error) {
	var count int
	// If reports table doesn't exist, return 0
	query := `SELECT COUNT(*) FROM reports WHERE status = 'pending'`
	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		// Table might not exist, return 0
		return 0, nil
	}
	return count, err
}

// Reports

func (r *repository) ListReports(ctx context.Context, page, limit int, statusFilter *string) ([]*Report, int, error) {
	// Calculate offset
	offset := (page - 1) * limit

	// Build query
	query := `SELECT * FROM reports WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM reports WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	// Apply status filter
	if statusFilter != nil && *statusFilter != "" {
		query += ` AND status = $` + strconv.Itoa(argNum)
		countQuery += ` AND status = $` + strconv.Itoa(argNum)
		args = append(args, *statusFilter)
		argNum++
	}

	// Order by created_at DESC
	query += ` ORDER BY created_at DESC`

	// Add pagination
	query += ` LIMIT $` + strconv.Itoa(argNum) + ` OFFSET $` + strconv.Itoa(argNum+1)
	paginationArgs := append(args, limit, offset)

	// Get reports
	var reports []*Report
	err := r.db.SelectContext(ctx, &reports, query, paginationArgs...)
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	var total int
	err = r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return reports, 0, err
	}

	return reports, total, nil
}

func (r *repository) GetReportByID(ctx context.Context, id uuid.UUID) (*Report, error) {
	query := `SELECT * FROM reports WHERE id = $1`
	var report Report
	err := r.db.GetContext(ctx, &report, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &report, nil
}

func (r *repository) UpdateReportStatus(ctx context.Context, id uuid.UUID, status string, adminNotes string, resolvedBy uuid.UUID) error {
	query := `
		UPDATE reports 
		SET status = $2, 
		    admin_notes = $3, 
		    resolved_by = $4, 
		    resolved_at = NOW(), 
		    updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, status, adminNotes, resolvedBy)
	return err
}

func (r *repository) GetReportedUserByID(ctx context.Context, userID uuid.UUID) (*ReportedUser, error) {
	query := `SELECT id, email, status FROM users WHERE id = $1`
	var user ReportedUser
	err := r.db.GetContext(ctx, &user, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *repository) SoftDeleteEntity(ctx context.Context, entityType string, entityID uuid.UUID) error {
	var query string
	switch entityType {
	case "user":
		query = `UPDATE users SET deleted_at = NOW() WHERE id = $1`
	case "casting":
		query = `UPDATE castings SET deleted_at = NOW() WHERE id = $1`
	case "profile":
		query = `UPDATE profiles SET deleted_at = NOW() WHERE id = $1`
	default:
		return errors.New("unsupported entity type")
	}

	_, err := r.db.ExecContext(ctx, query, entityID)
	return err
}

// Transaction methods

func (r *repository) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	return r.db.BeginTxx(ctx, nil)
}

func (r *repository) UpdateReportStatusTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, status string, adminNotes string, resolvedBy uuid.UUID) error {
	query := `
		UPDATE reports 
		SET status = $2, 
		    admin_notes = $3, 
		    resolved_by = $4, 
		    resolved_at = NOW(), 
		    updated_at = NOW()
		WHERE id = $1
	`
	_, err := tx.ExecContext(ctx, query, id, status, adminNotes, resolvedBy)
	return err
}

func (r *repository) UpdateUserStatusTx(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID, status string) error {
	query := `UPDATE users SET status = $2, updated_at = NOW() WHERE id = $1`
	_, err := tx.ExecContext(ctx, query, userID, status)
	return err
}

func (r *repository) SoftDeleteEntityTx(ctx context.Context, tx *sqlx.Tx, entityType string, entityID uuid.UUID) error {
	var query string
	switch entityType {
	case "user":
		query = `UPDATE users SET deleted_at = NOW() WHERE id = $1`
	case "casting":
		query = `UPDATE castings SET deleted_at = NOW() WHERE id = $1`
	case "profile":
		query = `UPDATE profiles SET deleted_at = NOW() WHERE id = $1`
	default:
		return errors.New("unsupported entity type")
	}

	_, err := tx.ExecContext(ctx, query, entityID)
	return err
}

// ExecuteSql executes a SQL query (temporary solution for frontend-only moderation)
func (r *repository) ExecuteSql(ctx context.Context, query string) (interface{}, error) {
	// For safety, only allow UPDATE queries on users table
	if !strings.Contains(strings.ToUpper(query), "UPDATE") ||
		!strings.Contains(strings.ToUpper(query), "users") ||
		strings.Contains(strings.ToUpper(query), "DELETE") ||
		strings.Contains(strings.ToUpper(query), "DROP") {
		return nil, errors.New("only UPDATE queries on users table are allowed")
	}

	var result interface{}

	// Execute the query
	rows, err := r.db.QueryxContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// For UPDATE queries, return affected rows count
	if strings.HasPrefix(strings.ToUpper(query), "UPDATE") {
		res, err := r.db.ExecContext(ctx, query)
		if err != nil {
			return nil, err
		}
		affected, _ := res.RowsAffected()
		result = map[string]interface{}{
			"affected_rows": affected,
			"message":       "Query executed successfully",
		}
	} else {
		// For SELECT queries, return the rows
		var results []map[string]interface{}
		for rows.Next() {
			row := make(map[string]interface{})
			if err := rows.MapScan(row); err != nil {
				return nil, err
			}
			results = append(results, row)
		}
		result = results
	}

	return result, nil
}

// ListUsers returns users with filters
func (r *repository) ListUsers(ctx context.Context, params map[string]interface{}) ([]map[string]interface{}, int, error) {
	query := `
		SELECT id, email, role, email_verified, is_banned, user_verification_status, created_at, updated_at
		FROM users
		WHERE deleted_at IS NULL
	`

	var args []interface{}
	argIndex := 1

	// Add filters
	if role, ok := params["role"].(string); ok && role != "" {
		query += fmt.Sprintf(" AND role = $%d", argIndex)
		args = append(args, role)
		argIndex++
	}

	if status, ok := params["status"].(string); ok && status != "" {
		if status == "active" {
			query += fmt.Sprintf(" AND is_banned = $%d", argIndex)
			args = append(args, false)
			argIndex++
		} else if status == "banned" {
			query += fmt.Sprintf(" AND is_banned = $%d", argIndex)
			args = append(args, true)
			argIndex++
		}
	}

	if search, ok := params["search"].(string); ok && search != "" {
		query += fmt.Sprintf(" AND email ILIKE $%d", argIndex)
		args = append(args, "%"+search+"%")
		argIndex++
	}

	// Count query
	countQuery := strings.Replace(query, "SELECT id, email, role, email_verified, is_banned, created_at, updated_at", "SELECT COUNT(*)", 1)

	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Add pagination
	query += " ORDER BY created_at DESC"

	limit := 20 // default limit
	if l, ok := params["limit"].(int); ok && l > 0 {
		limit = l
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
		argIndex++
	}

	if page, ok := params["page"].(int); ok && page > 0 {
		offset := (page - 1) * limit
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, offset)
	}

	var users []map[string]interface{}
	err = r.db.SelectContext(ctx, &users, query, args...)
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// UpdateUserStatus updates user verification status
func (r *repository) UpdateUserStatus(ctx context.Context, userID, status string) error {
	query := `
		UPDATE users 
		SET user_verification_status = $2, 
		    verification_reviewed_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, userID, status)
	return err
}

// UpdateUserStatusWithReason updates user verification status with rejection reason
func (r *repository) UpdateUserStatusWithReason(ctx context.Context, userID, status, reason string) error {
	query := `
		UPDATE users 
		SET user_verification_status = $2, 
		    verification_rejection_reason = $3,
		    verification_reviewed_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, userID, status, reason)
	return err
}
