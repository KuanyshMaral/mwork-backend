package photo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines photo data access interface
type Repository interface {
	Create(ctx context.Context, photo *Photo) error
	GetByID(ctx context.Context, id uuid.UUID) (*Photo, error)
	GetByKey(ctx context.Context, key string) (*Photo, error)
	ListByProfile(ctx context.Context, profileID uuid.UUID) ([]*Photo, error)
	CountByProfile(ctx context.Context, profileID uuid.UUID) (int, error)
	Delete(ctx context.Context, id uuid.UUID) error
	SetAvatar(ctx context.Context, profileID, photoID uuid.UUID) error
	ClearAvatar(ctx context.Context, profileID uuid.UUID) error
	UpdateSortOrder(ctx context.Context, photoID uuid.UUID, order int) error
	CountByProfileID(ctx context.Context, profileID string) (int, error)
}

type repository struct {
	db *sqlx.DB
}

// NewRepository creates new photo repository
func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, photo *Photo) error {
	query := `
		INSERT INTO photos (id, profile_id, key, url, original_name, mime_type, size_bytes, is_avatar, sort_order, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(ctx, query,
		photo.ID,
		photo.ProfileID,
		photo.Key,
		photo.URL,
		photo.OriginalName,
		photo.MimeType,
		photo.SizeBytes,
		photo.IsAvatar,
		photo.SortOrder,
		photo.CreatedAt,
	)
	return err
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Photo, error) {
	query := `SELECT * FROM photos WHERE id = $1`
	var photo Photo
	err := r.db.GetContext(ctx, &photo, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &photo, nil
}

func (r *repository) GetByKey(ctx context.Context, key string) (*Photo, error) {
	query := `SELECT * FROM photos WHERE key = $1`
	var photo Photo
	err := r.db.GetContext(ctx, &photo, query, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &photo, nil
}

func (r *repository) ListByProfile(ctx context.Context, profileID uuid.UUID) ([]*Photo, error) {
	query := `SELECT * FROM photos WHERE profile_id = $1 ORDER BY sort_order, created_at`
	var photos []*Photo
	err := r.db.SelectContext(ctx, &photos, query, profileID)
	return photos, err
}

func (r *repository) CountByProfile(ctx context.Context, profileID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM photos WHERE profile_id = $1`
	var count int
	err := r.db.GetContext(ctx, &count, query, profileID)
	return count, err
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM photos WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *repository) SetAvatar(ctx context.Context, profileID, photoID uuid.UUID) error {
	// First clear existing avatar
	clearQuery := `UPDATE photos SET is_avatar = false WHERE profile_id = $1 AND is_avatar = true`
	_, err := r.db.ExecContext(ctx, clearQuery, profileID)
	if err != nil {
		return err
	}

	// Set new avatar
	setQuery := `UPDATE photos SET is_avatar = true WHERE id = $1 AND profile_id = $2`
	_, err = r.db.ExecContext(ctx, setQuery, photoID, profileID)
	return err
}

func (r *repository) ClearAvatar(ctx context.Context, profileID uuid.UUID) error {
	query := `UPDATE photos SET is_avatar = false WHERE profile_id = $1`
	_, err := r.db.ExecContext(ctx, query, profileID)
	return err
}

func (r *repository) UpdateSortOrder(ctx context.Context, photoID uuid.UUID, order int) error {
	query := `UPDATE photos SET sort_order = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, photoID, order)
	return err
}

func (r *repository) CountByProfileID(ctx context.Context, profileID string) (int, error) {
	query := `SELECT COUNT(*) FROM photos WHERE profile_id = $1`
	var count int
	err := r.db.QueryRowContext(ctx, query, profileID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count photos: %w", err)
	}
	return count, nil
}
