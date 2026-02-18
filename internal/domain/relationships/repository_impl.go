package relationships

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type repository struct {
	db *sqlx.DB
}

// NewRepository creates new relationships repository
func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateBlock(ctx context.Context, block *BlockRelation) error {
	query := `
		INSERT INTO user_blocks (id, blocker_user_id, blocked_user_id, created_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.db.ExecContext(ctx, query, block.ID, block.BlockerUserID, block.BlockedUserID, block.CreatedAt)
	return err
}

func (r *repository) DeleteBlock(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	query := `DELETE FROM user_blocks WHERE blocker_user_id = $1 AND blocked_user_id = $2`
	_, err := r.db.ExecContext(ctx, query, blockerID, blockedID)
	return err
}

func (r *repository) HasBlocked(ctx context.Context, blockerID, blockedID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM user_blocks WHERE blocker_user_id = $1 AND blocked_user_id = $2)`
	var exists bool
	err := r.db.GetContext(ctx, &exists, query, blockerID, blockedID)
	return exists, err
}

func (r *repository) ListBlocks(ctx context.Context, userID uuid.UUID) ([]*BlockRelation, error) {
	query := `SELECT * FROM user_blocks WHERE blocker_user_id = $1 ORDER BY created_at DESC`
	var blocks []*BlockRelation
	err := r.db.SelectContext(ctx, &blocks, query, userID)
	return blocks, err
}
