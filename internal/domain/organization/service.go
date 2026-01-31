package organization

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/mwork/mwork-api/internal/domain/user"
)

// Service handles organization business logic
type Service struct {
	repo     *Repository
	userRepo user.Repository
}

// NewService creates organization service
func NewService(repo *Repository, userRepo user.Repository) *Service {
	return &Service{
		repo:     repo,
		userRepo: userRepo,
	}
}

// Create creates a new organization
func (s *Service) Create(ctx context.Context, userID uuid.UUID, req *CreateOrganizationRequest) (*Organization, error) {
	// Verify user exists
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return nil, ErrOrganizationNotFound
	}

	now := time.Now()
	org := &Organization{
		ID:                 uuid.New(),
		LegalName:          req.LegalName,
		BinIIN:             req.BinIIN,
		OrgType:            req.OrgType,
		VerificationStatus: VerificationNone,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if req.BrandName != "" {
		org.BrandName = sql.NullString{String: req.BrandName, Valid: true}
	}
	if req.City != "" {
		org.City = sql.NullString{String: req.City, Valid: true}
	}
	if req.Phone != "" {
		org.Phone = sql.NullString{String: req.Phone, Valid: true}
	}
	if req.Email != "" {
		org.Email = sql.NullString{String: req.Email, Valid: true}
	}
	if req.Website != "" {
		org.Website = sql.NullString{String: req.Website, Valid: true}
	}

	if err := s.repo.Create(ctx, org); err != nil {
		return nil, err
	}

	// Add creator as owner
	member := &OrganizationMember{
		ID:             uuid.New(),
		OrganizationID: org.ID,
		UserID:         userID,
		Role:           RoleOwner,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.repo.AddMember(ctx, member); err != nil {
		return nil, err
	}

	return org, nil
}

// GetByID returns organization by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Organization, error) {
	return s.repo.GetByID(ctx, id)
}

// GetMembers returns organization members
func (s *Service) GetMembers(ctx context.Context, orgID uuid.UUID) ([]*OrganizationMember, error) {
	return s.repo.GetMembers(ctx, orgID)
}

// AddMember invites a new member to organization
func (s *Service) AddMember(ctx context.Context, orgID uuid.UUID, inviterID uuid.UUID, targetUserID uuid.UUID, role MemberRole) (*OrganizationMember, error) {
	// Check if inviter has permission (must be owner or admin)
	inviterMember, err := s.repo.GetMemberByUserID(ctx, orgID, inviterID)
	if err != nil || inviterMember == nil {
		return nil, ErrNotAuthorized
	}
	if inviterMember.Role != RoleOwner && inviterMember.Role != RoleAdmin {
		return nil, ErrNotAuthorized
	}

	// Check if user already a member
	existing, _ := s.repo.GetMemberByUserID(ctx, orgID, targetUserID)
	if existing != nil {
		return nil, ErrMemberAlreadyExists
	}

	// Verify target user exists
	targetUser, err := s.userRepo.GetByID(ctx, targetUserID)
	if err != nil || targetUser == nil {
		return nil, ErrUserNotFound
	}

	now := time.Now()
	member := &OrganizationMember{
		ID:             uuid.New(),
		OrganizationID: orgID,
		UserID:         targetUserID,
		Role:           role,
		InvitedBy:      uuid.NullUUID{UUID: inviterID, Valid: true},
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.AddMember(ctx, member); err != nil {
		return nil, err
	}

	return member, nil
}

// UpdateMemberRole updates a member's role
func (s *Service) UpdateMemberRole(ctx context.Context, orgID uuid.UUID, updaterID uuid.UUID, memberID uuid.UUID, newRole MemberRole) error {
	// Check if updater has permission (must be owner)
	updaterMember, err := s.repo.GetMemberByUserID(ctx, orgID, updaterID)
	if err != nil || updaterMember == nil {
		return ErrNotAuthorized
	}
	if updaterMember.Role != RoleOwner {
		return ErrNotAuthorized
	}

	return s.repo.UpdateMemberRole(ctx, memberID, newRole)
}

// RemoveMember removes a member from organization
func (s *Service) RemoveMember(ctx context.Context, orgID uuid.UUID, removerID uuid.UUID, memberID uuid.UUID) error {
	// Check if remover has permission (must be owner or admin)
	removerMember, err := s.repo.GetMemberByUserID(ctx, orgID, removerID)
	if err != nil || removerMember == nil {
		return ErrNotAuthorized
	}
	if removerMember.Role != RoleOwner && removerMember.Role != RoleAdmin {
		return ErrNotAuthorized
	}

	// Get member to remove
	member, err := s.repo.GetMemberByID(ctx, memberID)
	if err != nil || member == nil {
		return ErrMemberNotFound
	}

	// Cannot remove owner
	if member.Role == RoleOwner {
		return ErrCannotRemoveOwner
	}

	return s.repo.RemoveMember(ctx, memberID)
}

// Follow allows a user to follow an organization
func (s *Service) Follow(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) error {
	// Check if org exists
	org, err := s.repo.GetByID(ctx, orgID)
	if err != nil || org == nil {
		return ErrOrganizationNotFound
	}

	// Check if already following
	isFollowing, _ := s.repo.IsFollowing(ctx, orgID, userID)
	if isFollowing {
		return ErrAlreadyFollowing
	}

	now := time.Now()
	follower := &AgencyFollower{
		ID:             uuid.New(),
		OrganizationID: orgID,
		FollowerUserID: userID,
		CreatedAt:      now,
	}

	return s.repo.AddFollower(ctx, follower)
}

// Unfollow allows a user to unfollow an organization
func (s *Service) Unfollow(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) error {
	return s.repo.RemoveFollower(ctx, orgID, userID)
}

// CheckFollowing checks if user is following organization
func (s *Service) CheckFollowing(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) (bool, error) {
	return s.repo.IsFollowing(ctx, orgID, userID)
}

// GetFollowers returns organization followers
func (s *Service) GetFollowers(ctx context.Context, orgID uuid.UUID) ([]*AgencyFollower, error) {
	return s.repo.GetFollowers(ctx, orgID)
}
