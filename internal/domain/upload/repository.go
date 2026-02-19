package upload

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines data access for uploads.
type Repository interface {
	Create(ctx context.Context, upload *Upload) error
	GetByID(ctx context.Context, id uuid.UUID) (*Upload, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ListByAuthor(ctx context.Context, authorID uuid.UUID) ([]*Upload, error)
}

type repository struct {
	db *sqlx.DB
}

// NewRepository creates upload repository.
func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

const selectColumns = `id, author_id, file_path, original_name, mime_type, size_bytes, created_at`

func (r *repository) Create(ctx context.Context, u *Upload) error {
	query := `
		INSERT INTO uploads (id, author_id, file_path, original_name, mime_type, size_bytes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		u.ID, u.AuthorID, u.FilePath, u.OriginalName, u.MimeType, u.SizeBytes, u.CreatedAt,
	)
	return err
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Upload, error) {
	query := `SELECT ` + selectColumns + ` FROM uploads WHERE id = $1`
	var u Upload
	if err := r.db.GetContext(ctx, &u, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM uploads WHERE id = $1`, id)
	return err
}

func (r *repository) ListByAuthor(ctx context.Context, authorID uuid.UUID) ([]*Upload, error) {
	query := `SELECT ` + selectColumns + ` FROM uploads WHERE author_id = $1 ORDER BY created_at DESC`
	var uploads []*Upload
	if err := r.db.SelectContext(ctx, &uploads, query, authorID); err != nil {
		return nil, err
	}
	return uploads, nil
}
