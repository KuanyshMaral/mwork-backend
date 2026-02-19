package attachment

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines data access for attachments.
type Repository interface {
	// Create adds a new attachment record.
	Create(ctx context.Context, a *Attachment) error

	// ListByTarget returns all attachments for an entity, ordered by sort_order.
	ListByTarget(ctx context.Context, targetType TargetType, targetID uuid.UUID) ([]*Attachment, error)

	// GetByID returns a single attachment.
	GetByID(ctx context.Context, id uuid.UUID) (*Attachment, error)

	// Delete removes an attachment by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// UpdateSortOrder updates the sort order of a single attachment.
	UpdateSortOrder(ctx context.Context, id uuid.UUID, sortOrder int) error

	// CountByTarget returns how many attachments a target has.
	CountByTarget(ctx context.Context, targetType TargetType, targetID uuid.UUID) (int, error)
}

type repository struct {
	db *sqlx.DB
}

// NewRepository creates a new attachment repository.
func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

const selectCols = `id, upload_id, target_id, target_type, sort_order, metadata, created_at`

func (r *repository) Create(ctx context.Context, a *Attachment) error {
	query := `
		INSERT INTO attachments (id, upload_id, target_id, target_type, sort_order, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		a.ID, a.UploadID, a.TargetID, a.TargetType, a.SortOrder, a.Metadata, a.CreatedAt,
	)
	return err
}

func (r *repository) ListByTarget(ctx context.Context, targetType TargetType, targetID uuid.UUID) ([]*Attachment, error) {
	query := `SELECT ` + selectCols + `
		FROM attachments
		WHERE target_type = $1 AND target_id = $2
		ORDER BY sort_order ASC, created_at ASC`
	var list []*Attachment
	if err := r.db.SelectContext(ctx, &list, query, targetType, targetID); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Attachment, error) {
	query := `SELECT ` + selectCols + ` FROM attachments WHERE id = $1`
	var a Attachment
	if err := r.db.GetContext(ctx, &a, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM attachments WHERE id = $1`, id)
	return err
}

func (r *repository) UpdateSortOrder(ctx context.Context, id uuid.UUID, sortOrder int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE attachments SET sort_order = $1 WHERE id = $2`,
		sortOrder, id)
	return err
}

func (r *repository) CountByTarget(ctx context.Context, targetType TargetType, targetID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM attachments WHERE target_type = $1 AND target_id = $2`,
		targetType, targetID)
	return count, err
}
