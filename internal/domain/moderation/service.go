package moderation

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/user"
)

// Service handles moderation business logic
type Service struct {
	repo     Repository
	userRepo user.Repository
}

// NewService creates moderation service
func NewService(repo Repository, userRepo user.Repository) *Service {
	return &Service{
		repo:     repo,
		userRepo: userRepo,
	}
}

// BlockUser blocks a user
func (s *Service) BlockUser(ctx context.Context, blockerID uuid.UUID, req *BlockUserRequest) error {
	// Validate: cannot block self
	if blockerID == req.UserID {
		return ErrCannotBlockSelf
	}

	// Check if user exists
	targetUser, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil || targetUser == nil {
		return ErrBlockNotFound
	}

	// Check if already blocked
	existing, err := s.repo.GetBlock(ctx, blockerID, req.UserID)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrAlreadyBlocked
	}

	// Create block
	block := &UserBlock{
		ID:            uuid.New(),
		BlockerUserID: blockerID,
		BlockedUserID: req.UserID,
		CreatedAt:     time.Now(),
	}

	return s.repo.CreateBlock(ctx, block)
}

// UnblockUser removes a block
func (s *Service) UnblockUser(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	return s.repo.DeleteBlock(ctx, blockerID, blockedID)
}

// ListBlocks returns all blocks for a user
func (s *Service) ListBlocks(ctx context.Context, userID uuid.UUID) ([]*UserBlock, error) {
	return s.repo.ListBlocksByUser(ctx, userID)
}

// IsBlocked checks if either user has blocked the other
func (s *Service) IsBlocked(ctx context.Context, user1, user2 uuid.UUID) (bool, error) {
	return s.repo.IsBlocked(ctx, user1, user2)
}

// CreateReport creates a new user report
func (s *Service) CreateReport(ctx context.Context, reporterID uuid.UUID, req *CreateReportRequest) (*UserReport, error) {
	// Validate: cannot report self
	if reporterID == req.ReportedUserID {
		return nil, ErrCannotReportSelf
	}

	// Check if reported user exists
	reportedUser, err := s.userRepo.GetByID(ctx, req.ReportedUserID)
	if err != nil || reportedUser == nil {
		return nil, ErrReportNotFound
	}

	// Create report
	report := &UserReport{
		ID:             uuid.New(),
		ReporterUserID: reporterID,
		ReportedUserID: req.ReportedUserID,
		Reason:         req.Reason,
		Status:         ReportStatusPending,
		CreatedAt:      time.Now(),
	}

	if req.RoomID != nil {
		report.RoomID = uuid.NullUUID{UUID: *req.RoomID, Valid: true}
	}

	if req.MessageID != nil {
		report.MessageID = uuid.NullUUID{UUID: *req.MessageID, Valid: true}
	}

	if req.Description != "" {
		report.Description.String = req.Description
		report.Description.Valid = true
	}

	if err := s.repo.CreateReport(ctx, report); err != nil {
		return nil, err
	}

	return report, nil
}

// ListMyReports returns reports created by the user
func (s *Service) ListMyReports(ctx context.Context, userID uuid.UUID) ([]*UserReport, error) {
	return s.repo.ListReportsByReporter(ctx, userID)
}

// ListReports returns all reports (admin function)
func (s *Service) ListReports(ctx context.Context, filter *ListReportsFilter) ([]*UserReport, error) {
	return s.repo.ListReports(ctx, filter)
}

// GetReport returns a specific report (admin function)
func (s *Service) GetReport(ctx context.Context, id uuid.UUID) (*UserReport, error) {
	report, err := s.repo.GetReportByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if report == nil {
		return nil, ErrReportNotFound
	}
	return report, nil
}

// ResolveReport handles admin action on a report
func (s *Service) ResolveReport(ctx context.Context, reportID uuid.UUID, req *ResolveReportRequest) error {
	// Get report
	report, err := s.repo.GetReportByID(ctx, reportID)
	if err != nil {
		return err
	}
	if report == nil {
		return ErrReportNotFound
	}

	// Determine new status based on action
	var newStatus ReportStatus
	switch req.Action {
	case "warn", "suspend", "delete":
		newStatus = ReportStatusResolved
	case "dismiss":
		newStatus = ReportStatusDismissed
	default:
		return ErrInvalidReportStatus
	}

	// Update report
	return s.repo.UpdateReportStatus(ctx, reportID, newStatus, req.Notes)
}

// CountReports returns total report count (admin function)
func (s *Service) CountReports(ctx context.Context, filter *ListReportsFilter) (int, error) {
	return s.repo.CountReports(ctx, filter)
}
