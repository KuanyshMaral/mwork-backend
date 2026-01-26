package admin

import (
	"context"
	"database/sql"
	"encoding/json"
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
