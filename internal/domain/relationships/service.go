package relationships

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Service handles user relationships business logic
type Service struct {
	repo Repository
}

// NewService creates new relationships service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// HasBlocked checks if blockerID has blocked targetID
func (s *Service) HasBlocked(ctx context.Context, blockerID, targetID uuid.UUID) (bool, error) {
	return s.repo.HasBlocked(ctx, blockerID, targetID)
}

// BlockUser blocks a user
func (s *Service) BlockUser(ctx context.Context, blockerID, targetID uuid.UUID) error {
	block := &BlockRelation{
		ID:            uuid.New(),
		BlockerUserID: blockerID,
		BlockedUserID: targetID,
		CreatedAt:     time.Now(),
	}
	return s.repo.CreateBlock(ctx, block)
}

// UnblockUser unblocks a user
func (s *Service) UnblockUser(ctx context.Context, blockerID, targetID uuid.UUID) error {
	return s.repo.DeleteBlock(ctx, blockerID, targetID)
}

// ListMyBlocks returns all users blocked by the given user
func (s *Service) ListMyBlocks(ctx context.Context, userID uuid.UUID) ([]*BlockRelation, error) {
	return s.repo.ListBlocks(ctx, userID)
}
