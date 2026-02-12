package lead

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/organization"
	"github.com/mwork/mwork-api/internal/domain/user"
	"github.com/mwork/mwork-api/internal/pkg/password"
)

// Service handles lead business logic
type Service struct {
	repo     Repository
	orgRepo  *organization.Repository
	userRepo user.Repository
}

// NewService creates lead service
func NewService(repo Repository, orgRepo *organization.Repository, userRepo user.Repository) *Service {
	return &Service{
		repo:     repo,
		orgRepo:  orgRepo,
		userRepo: userRepo,
	}
}

// SubmitLead creates a new employer lead (public endpoint)
func (s *Service) SubmitLead(ctx context.Context, req *CreateLeadRequest, ip string, userAgent string) (*EmployerLead, error) {
	now := time.Now()

	lead := &EmployerLead{
		ID:              uuid.New(),
		ContactName:     req.ContactName,
		ContactEmail:    req.ContactEmail,
		ContactPhone:    req.ContactPhone,
		ContactPosition: sql.NullString{String: req.ContactPosition, Valid: req.ContactPosition != ""},
		CompanyName:     req.CompanyName,
		BinIIN:          sql.NullString{String: req.BinIIN, Valid: req.BinIIN != ""},
		OrgType:         organization.OrgType(req.OrgType),
		Website:         sql.NullString{String: req.Website, Valid: req.Website != ""},
		Industry:        sql.NullString{String: req.Industry, Valid: req.Industry != ""},
		EmployeesCount:  sql.NullString{String: req.EmployeesCount, Valid: req.EmployeesCount != ""},
		UseCase:         sql.NullString{String: req.UseCase, Valid: req.UseCase != ""},
		HowFoundUs:      sql.NullString{String: req.HowFoundUs, Valid: req.HowFoundUs != ""},
		Status:          StatusNew,
		Source:          sql.NullString{String: "website", Valid: true},
		UTMSource:       sql.NullString{String: req.UTMSource, Valid: req.UTMSource != ""},
		UTMMedium:       sql.NullString{String: req.UTMMedium, Valid: req.UTMMedium != ""},
		UTMCampaign:     sql.NullString{String: req.UTMCampaign, Valid: req.UTMCampaign != ""},
		IPAddress:       sql.NullString{String: ip, Valid: ip != ""},
		UserAgent:       sql.NullString{String: userAgent, Valid: userAgent != ""},
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.Create(ctx, lead); err != nil {
		return nil, err
	}

	return lead, nil
}

// GetByID returns lead by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*EmployerLead, error) {
	lead, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if lead == nil {
		return nil, ErrLeadNotFound
	}
	return lead, nil
}

// ListLeads returns leads with optional status filter
func (s *Service) ListLeads(ctx context.Context, status *Status, limit, offset int) ([]*EmployerLead, int, error) {
	return s.repo.List(ctx, status, limit, offset)
}

// UpdateStatus updates lead status
func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, status Status, notes, reason string) error {
	lead, err := s.repo.GetByID(ctx, id)
	if err != nil || lead == nil {
		return ErrLeadNotFound
	}

	if lead.IsConverted() {
		return ErrAlreadyConverted
	}

	return s.repo.UpdateStatus(ctx, id, status, notes, reason)
}

// MarkContacted marks lead as contacted
func (s *Service) MarkContacted(ctx context.Context, id uuid.UUID) error {
	lead, err := s.repo.GetByID(ctx, id)
	if err != nil || lead == nil {
		return ErrLeadNotFound
	}

	if lead.Status == StatusNew {
		_ = s.repo.UpdateStatus(ctx, id, StatusContacted, "", "")
	}

	return s.repo.MarkContacted(ctx, id)
}

// Assign assigns lead to admin
func (s *Service) Assign(ctx context.Context, id uuid.UUID, adminID uuid.UUID, priority int) error {
	lead, err := s.repo.GetByID(ctx, id)
	if err != nil || lead == nil {
		return ErrLeadNotFound
	}

	return s.repo.Assign(ctx, id, adminID, priority)
}

// Convert converts lead to employer account
func (s *Service) Convert(ctx context.Context, leadID uuid.UUID, req *ConvertRequest) (*user.User, *organization.Organization, error) {
	lead, err := s.repo.GetByID(ctx, leadID)
	if err != nil || lead == nil {
		return nil, nil, ErrLeadNotFound
	}

	if lead.IsConverted() {
		return nil, nil, ErrAlreadyConverted
	}

	if lead.Status != StatusQualified && lead.Status != StatusContacted {
		return nil, nil, ErrCannotConvert
	}

	// Check if BIN already exists
	existingOrg, _ := s.orgRepo.GetByBIN(ctx, req.BinIIN)
	if existingOrg != nil {
		return nil, nil, organization.ErrBINAlreadyExists
	}

	// Check if email already exists
	existingUser, _ := s.userRepo.GetByEmail(ctx, lead.ContactEmail)
	if existingUser != nil {
		return nil, nil, user.ErrEmailExists
	}

	now := time.Now()

	// Create organization
	org := &organization.Organization{
		ID:                 uuid.New(),
		LegalName:          req.LegalName,
		BinIIN:             req.BinIIN,
		OrgType:            organization.OrgType(req.OrgType),
		LegalAddress:       sql.NullString{String: req.LegalAddress, Valid: req.LegalAddress != ""},
		Email:              sql.NullString{String: lead.ContactEmail, Valid: true},
		Phone:              sql.NullString{String: lead.ContactPhone, Valid: true},
		VerificationStatus: organization.VerificationVerified, // Pre-verified since manually converted
		VerifiedAt:         sql.NullTime{Time: now, Valid: true},
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if lead.Website.Valid {
		org.Website = lead.Website
	}

	if err := s.orgRepo.Create(ctx, org); err != nil {
		return nil, nil, err
	}

	// Create user with hashed password
	hashedPwd, err := password.Hash(req.Password)
	if err != nil {
		return nil, nil, err
	}

	newUser := &user.User{
		ID:            uuid.New(),
		Email:         lead.ContactEmail,
		PasswordHash:  hashedPwd,
		Role:          user.RoleEmployer,
		EmailVerified: true, // Pre-verified
		IsVerified:    true, // Allow immediate login after manual admin conversion
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.userRepo.Create(ctx, newUser); err != nil {
		// Rollback org creation
		_ = s.orgRepo.Delete(ctx, org.ID)
		return nil, nil, err
	}

	// Mark lead as converted
	_ = s.repo.MarkConverted(ctx, leadID, newUser.ID, org.ID)

	return newUser, org, nil
}

// GetStats returns lead statistics
func (s *Service) GetStats(ctx context.Context) (map[Status]int, error) {
	return s.repo.CountByStatus(ctx)
}
