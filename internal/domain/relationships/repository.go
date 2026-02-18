package relationships

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines relationships data access interface
type Repository interface {
	CreateBlock(ctx context.Context, block *BlockRelation) error
	DeleteBlock(ctx context.Context, blockerID, blockedID uuid.UUID) error
	HasBlocked(ctx context.Context, blockerID, blockedID uuid.UUID) (bool, error)
	ListBlocks(ctx context.Context, userID uuid.UUID) ([]*BlockRelation, error)
}
