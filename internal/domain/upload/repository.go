package upload

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines upload data access interface
type Repository interface {
	Create(ctx context.Context, upload *Upload) error
	GetByID(ctx context.Context, id uuid.UUID) (*Upload, error)
	Update(ctx context.Context, upload *Upload) error
	UpdateStaged(ctx context.Context, upload *Upload) error
	MarkCommitted(ctx context.Context, id uuid.UUID, size int64, permanentKey, permanentURL string, committedAt time.Time) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByUser(ctx context.Context, userID uuid.UUID, category Category) ([]*Upload, error)
	ListExpired(ctx context.Context, before time.Time) ([]*Upload, error)
	DeleteExpired(ctx context.Context, before time.Time) (int, error)
}

type repository struct {
	db *sqlx.DB
}

// NewRepository creates upload repository
func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, upload *Upload) error {
	query := `
		INSERT INTO uploads (
			id, user_id, category, status,
			original_name, mime_type, size,
			staging_key, permanent_key, permanent_url,
			width, height, error_message,
			created_at, committed_at, expires_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7,
			$8, $9, $10,
			$11, $12, $13,
			$14, $15, $16
		)
	`
	_, err := r.db.ExecContext(ctx, query,
		upload.ID, upload.UserID, upload.Category, upload.Status,
		upload.OriginalName, upload.MimeType, upload.Size,
		upload.StagingKey, upload.PermanentKey, upload.PermanentURL,
		upload.Width, upload.Height, upload.ErrorMessage,
		upload.CreatedAt, upload.CommittedAt, upload.ExpiresAt,
	)
	return err
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Upload, error) {
	query := `SELECT * FROM uploads WHERE id = $1 AND status != 'deleted'`
	var upload Upload
	err := r.db.GetContext(ctx, &upload, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &upload, nil
}

func (r *repository) Update(ctx context.Context, upload *Upload) error {
	query := `
		UPDATE uploads SET
			status = $2,
			permanent_key = $3,
			permanent_url = $4,
			width = $5,
			height = $6,
			error_message = $7,
			committed_at = $8
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		upload.ID,
		upload.Status,
		upload.PermanentKey,
		upload.PermanentURL,
		upload.Width,
		upload.Height,
		upload.ErrorMessage,
		upload.CommittedAt,
	)
	return err
}

func (r *repository) UpdateStaged(ctx context.Context, upload *Upload) error {
	query := `
		UPDATE uploads SET
			category = $2,
			status = $3,
			original_name = $4,
			mime_type = $5,
			size = $6,
			staging_key = $7,
			expires_at = $8,
			permanent_key = NULL,
			permanent_url = NULL,
			committed_at = NULL,
			error_message = NULL
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		upload.ID,
		upload.Category,
		upload.Status,
		upload.OriginalName,
		upload.MimeType,
		upload.Size,
		upload.StagingKey,
		upload.ExpiresAt,
	)
	return err
}

func (r *repository) MarkCommitted(ctx context.Context, id uuid.UUID, size int64, permanentKey, permanentURL string, committedAt time.Time) error {
	query := `
		UPDATE uploads SET
			status = 'committed',
			size = $2,
			permanent_key = $3,
			permanent_url = $4,
			committed_at = $5
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, size, permanentKey, permanentURL, committedAt)
	return err
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE uploads SET status = 'deleted' WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *repository) ListByUser(ctx context.Context, userID uuid.UUID, category Category) ([]*Upload, error) {
	query := `
		SELECT * FROM uploads 
		WHERE user_id = $1 
		AND ($2 = '' OR category = $2)
		AND status = 'committed'
		ORDER BY created_at DESC
	`
	var uploads []*Upload
	err := r.db.SelectContext(ctx, &uploads, query, userID, category)
	return uploads, err
}

func (r *repository) ListExpired(ctx context.Context, before time.Time) ([]*Upload, error) {
	query := `
		SELECT * FROM uploads 
		WHERE status = 'staged' 
		AND expires_at < $1
	`
	var uploads []*Upload
	err := r.db.SelectContext(ctx, &uploads, query, before)
	return uploads, err
}

func (r *repository) DeleteExpired(ctx context.Context, before time.Time) (int, error) {
	query := `
		DELETE FROM uploads 
		WHERE status = 'staged' 
		AND expires_at < $1
		RETURNING id
	`
	result, err := r.db.ExecContext(ctx, query, before)
	if err != nil {
		return 0, err
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}
