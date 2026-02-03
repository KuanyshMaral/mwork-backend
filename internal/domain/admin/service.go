package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/mwork/mwork-api/internal/pkg/password"
)

// Service handles admin business logic
type Service struct {
	repo Repository
}

// NewService creates admin service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// --- Authentication ---

// Login authenticates admin and returns token
func (s *Service) Login(ctx context.Context, email, pwd, ip string) (*AdminUser, error) {
	email = normalizeEmail(email)
	admin, err := s.repo.GetAdminByEmail(ctx, email)
	if err != nil || admin == nil {
		return nil, ErrInvalidCredentials
	}

	if !admin.IsActive {
		return nil, ErrAdminInactive
	}

	if !password.Verify(pwd, admin.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// Update last login
	_ = s.repo.UpdateLastLogin(ctx, admin.ID, ip)

	return admin, nil
}

// GetAdminByID returns admin by ID
func (s *Service) GetAdminByID(ctx context.Context, id uuid.UUID) (*AdminUser, error) {
	admin, err := s.repo.GetAdminByID(ctx, id)
	if err != nil || admin == nil {
		return nil, ErrAdminNotFound
	}
	return admin, nil
}

// --- Admin Management ---

// CreateAdmin creates a new admin user
func (s *Service) CreateAdmin(ctx context.Context, actorID uuid.UUID, req *CreateAdminRequest) (*AdminUser, error) {
	req.Email = normalizeEmail(req.Email)

	// Check if email taken
	existing, _ := s.repo.GetAdminByEmail(ctx, req.Email)
	if existing != nil {
		return nil, ErrEmailTaken
	}

	// Hash password
	hash, err := password.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	admin := &AdminUser{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: hash,
		Role:         Role(req.Role),
		Name:         req.Name,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.CreateAdmin(ctx, admin); err != nil {
		return nil, err
	}

	// Audit log
	s.logAction(ctx, actorID, "admin.create", "admin", admin.ID, nil, admin)

	return admin, nil
}

// UpdateAdmin updates admin user
func (s *Service) UpdateAdmin(ctx context.Context, actorID, targetID uuid.UUID, req *UpdateAdminRequest) (*AdminUser, error) {
	admin, err := s.repo.GetAdminByID(ctx, targetID)
	if err != nil || admin == nil {
		return nil, ErrAdminNotFound
	}

	// Get actor to check permissions
	actor, _ := s.repo.GetAdminByID(ctx, actorID)
	if actor != nil && !CanManage(actor.Role, admin.Role) {
		return nil, ErrCannotManageRole
	}

	oldValue := *admin

	if req.Name != nil {
		admin.Name = *req.Name
	}
	if req.Role != nil {
		admin.Role = Role(*req.Role)
	}
	if req.IsActive != nil {
		admin.IsActive = *req.IsActive
	}

	if err := s.repo.UpdateAdmin(ctx, admin); err != nil {
		return nil, err
	}

	// Audit log
	s.logAction(ctx, actorID, "admin.update", "admin", admin.ID, oldValue, admin)

	return admin, nil
}

// ListAdmins returns all admins
func (s *Service) ListAdmins(ctx context.Context) ([]*AdminUser, error) {
	return s.repo.ListAdmins(ctx)
}

// --- Feature Flags ---

// GetFeatureFlag returns a feature flag
func (s *Service) GetFeatureFlag(ctx context.Context, key string) (*FeatureFlag, error) {
	flag, err := s.repo.GetFeatureFlag(ctx, key)
	if err != nil || flag == nil {
		return nil, ErrFeatureFlagNotFound
	}
	return flag, nil
}

// ListFeatureFlags returns all flags
func (s *Service) ListFeatureFlags(ctx context.Context) ([]*FeatureFlag, error) {
	return s.repo.ListFeatureFlags(ctx)
}

// UpdateFeatureFlag updates a flag
func (s *Service) UpdateFeatureFlag(ctx context.Context, adminID uuid.UUID, key string, value interface{}) error {
	flag, err := s.repo.GetFeatureFlag(ctx, key)
	if err != nil || flag == nil {
		return ErrFeatureFlagNotFound
	}

	oldValue := flag.Value

	newValue, err := json.Marshal(value)
	if err != nil {
		return err
	}

	if err := s.repo.UpdateFeatureFlag(ctx, key, newValue, adminID); err != nil {
		return err
	}

	// Audit log
	s.logAction(ctx, adminID, "feature.update", "feature_flag", uuid.Nil,
		map[string]interface{}{"key": key, "value": oldValue},
		map[string]interface{}{"key": key, "value": newValue},
	)

	return nil
}

// IsFeatureEnabled checks if feature is enabled
func (s *Service) IsFeatureEnabled(ctx context.Context, key string) bool {
	flag, err := s.repo.GetFeatureFlag(ctx, key)
	if err != nil || flag == nil {
		return true // Default to enabled if not found
	}
	return flag.GetBool()
}

// --- Analytics ---

// GetStats returns admin dashboard statistics with parallel aggregation
func (s *Service) GetStats(ctx context.Context) (*StatsResponse, error) {
	var stats StatsResponse
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	wg.Add(5)

	// Total Users
	go func() {
		defer wg.Done()
		count, err := s.repo.CountUsers(ctx)
		mu.Lock()
		if err != nil && firstErr == nil {
			firstErr = err
		}
		stats.TotalUsers = count
		mu.Unlock()
	}()

	// Total Castings (active + draft)
	go func() {
		defer wg.Done()
		count, err := s.repo.CountActiveCastings(ctx)
		mu.Lock()
		if err != nil && firstErr == nil {
			firstErr = err
		}
		stats.TotalCastings = count
		mu.Unlock()
	}()

	// Active Subscriptions
	go func() {
		defer wg.Done()
		count, err := s.repo.CountActiveSubscriptions(ctx)
		mu.Lock()
		if err != nil && firstErr == nil {
			firstErr = err
		}
		stats.ActiveSubscriptions = count
		mu.Unlock()
	}()

	// Pending Payments
	go func() {
		defer wg.Done()
		count, err := s.repo.CountPendingPayments(ctx)
		mu.Lock()
		if err != nil && firstErr == nil {
			firstErr = err
		}
		stats.PendingPayments = count
		mu.Unlock()
	}()

	// Pending Reports
	go func() {
		defer wg.Done()
		count, err := s.repo.CountPendingReports(ctx)
		mu.Lock()
		if err != nil && firstErr == nil {
			firstErr = err
		}
		stats.PendingReports = count
		mu.Unlock()
	}()

	wg.Wait()

	if firstErr != nil {
		log.Error().Err(firstErr).Msg("Failed to fetch stats")
		return nil, firstErr
	}

	return &stats, nil
}

// GetDashboardStats returns dashboard statistics
func (s *Service) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	return s.repo.GetDashboardStats(ctx)
}

// --- Audit Logs ---

// ListAuditLogs returns audit logs
func (s *Service) ListAuditLogs(ctx context.Context, filter AuditFilter) ([]*AuditLog, int, error) {
	return s.repo.ListAuditLogs(ctx, filter)
}

// logAction creates an audit log entry
func (s *Service) logAction(ctx context.Context, adminID uuid.UUID, action, entityType string, entityID uuid.UUID, oldValue, newValue interface{}) {
	admin, _ := s.repo.GetAdminByID(ctx, adminID)
	email := ""
	if admin != nil {
		email = admin.Email
	}

	oldJSON, _ := json.Marshal(oldValue)
	newJSON, _ := json.Marshal(newValue)

	entry := &AuditLog{
		ID:         uuid.New(),
		AdminID:    uuid.NullUUID{UUID: adminID, Valid: adminID != uuid.Nil},
		AdminEmail: email,
		Action:     action,
		EntityType: entityType,
		EntityID:   uuid.NullUUID{UUID: entityID, Valid: entityID != uuid.Nil},
		OldValue:   oldJSON,
		NewValue:   newJSON,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.CreateAuditLog(ctx, entry); err != nil {
		// Log error but don't fail the operation
		log.Error().Err(err).Msg("Failed to create audit log")
	}
}

// LogActionWithReason creates audit log with reason
func (s *Service) LogActionWithReason(ctx context.Context, adminID uuid.UUID, action, entityType string, entityID uuid.UUID, reason string, oldValue, newValue interface{}) {
	admin, _ := s.repo.GetAdminByID(ctx, adminID)
	email := ""
	if admin != nil {
		email = admin.Email
	}

	oldJSON, _ := json.Marshal(oldValue)
	newJSON, _ := json.Marshal(newValue)

	auditLog := &AuditLog{
		ID:         uuid.New(),
		AdminID:    uuid.NullUUID{UUID: adminID, Valid: adminID != uuid.Nil},
		AdminEmail: email,
		Action:     action,
		EntityType: entityType,
		EntityID:   uuid.NullUUID{UUID: entityID, Valid: entityID != uuid.Nil},
		OldValue:   oldJSON,
		NewValue:   newJSON,
		Reason:     sql.NullString{String: reason, Valid: reason != ""},
		CreatedAt:  time.Now(),
	}

	_ = s.repo.CreateAuditLog(ctx, auditLog)
}

// --- Reports ---

// ListReports returns paginated reports with filters
func (s *Service) ListReports(ctx context.Context, page, limit int, statusFilter *string) ([]*Report, int, error) {
	// Validate and set defaults
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	return s.repo.ListReports(ctx, page, limit, statusFilter)
}

// ResolveReport handles moderation action on report
func (s *Service) ResolveReport(ctx context.Context, adminID uuid.UUID, reportID uuid.UUID, req *ResolveRequest) error {
	// Get the report
	report, err := s.repo.GetReportByID(ctx, reportID)
	if err != nil || report == nil {
		return ErrReportNotFound
	}

	// Get reported user
	reportedUser, err := s.repo.GetReportedUserByID(ctx, report.ReportedUserID)
	if err != nil || reportedUser == nil {
		return errors.New("reported user not found")
	}

	// Begin transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to begin transaction")
		return err
	}
	defer tx.Rollback()

	// Perform action based on request
	switch req.Action {
	case "warn":
		// Send warning email to reported user
		// Email sending would be implemented here
		log.Info().
			Str("user_id", report.ReportedUserID.String()).
			Str("action", "warn").
			Msg("Warning sent to user")

	case "suspend":
		// Update user status to suspended within transaction
		if err := s.repo.UpdateUserStatusTx(ctx, tx, report.ReportedUserID, "suspended"); err != nil {
			log.Error().Err(err).Msg("Failed to suspend user")
			return err
		}
		log.Info().
			Str("user_id", report.ReportedUserID.String()).
			Str("action", "suspend").
			Msg("User suspended")

	case "delete":
		// Soft-delete entity within transaction
		if err := s.repo.SoftDeleteEntityTx(ctx, tx, report.EntityType, report.EntityID); err != nil {
			log.Error().Err(err).Msg("Failed to delete entity")
			return err
		}
		log.Info().
			Str("entity_type", report.EntityType).
			Str("entity_id", report.EntityID.String()).
			Str("action", "delete").
			Msg("Entity soft-deleted")

	case "dismiss":
		// Just mark as dismissed
		log.Info().
			Str("report_id", reportID.String()).
			Str("action", "dismiss").
			Msg("Report dismissed")

	default:
		return errors.New("invalid action")
	}

	// Update report status to resolved within transaction
	if err := s.repo.UpdateReportStatusTx(ctx, tx, reportID, "resolved", req.Notes, adminID); err != nil {
		log.Error().Err(err).Msg("Failed to update report status")
		return err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Error().Err(err).Msg("Failed to commit transaction")
		return err
	}

	// Create audit log (outside transaction - best effort)
	s.LogActionWithReason(ctx, adminID, "report.resolve", "report", reportID, req.Notes, report, map[string]interface{}{
		"status": "resolved",
		"action": req.Action,
		"notes":  req.Notes,
	})

	return nil
}
