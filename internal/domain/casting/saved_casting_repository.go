package casting

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// SavedCastingRepository handles saved casting database operations
type SavedCastingRepository struct {
	db *sqlx.DB
}

// NewSavedCastingRepository creates a new repository
func NewSavedCastingRepository(db *sqlx.DB) *SavedCastingRepository {
	return &SavedCastingRepository{db: db}
}

// ErrAlreadySaved is returned when casting is already saved
var ErrAlreadySaved = errors.New("casting already saved")

// Save adds a casting to user's favorites
func (r *SavedCastingRepository) Save(ctx context.Context, userID, castingID uuid.UUID) error {
	query := `
		INSERT INTO saved_castings (id, user_id, casting_id, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id, casting_id) DO NOTHING
	`

	result, err := r.db.ExecContext(ctx, query, uuid.New(), userID, castingID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrAlreadySaved
	}
	return nil
}

// Unsave removes a casting from user's favorites
func (r *SavedCastingRepository) Unsave(ctx context.Context, userID, castingID uuid.UUID) error {
	query := `DELETE FROM saved_castings WHERE user_id = $1 AND casting_id = $2`

	result, err := r.db.ExecContext(ctx, query, userID, castingID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// IsSaved checks if a casting is saved by user
func (r *SavedCastingRepository) IsSaved(ctx context.Context, userID, castingID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM saved_castings WHERE user_id = $1 AND casting_id = $2)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, userID, castingID)
	return exists, err
}

// GetSavedCastingIDs returns list of saved casting IDs for a user
func (r *SavedCastingRepository) GetSavedCastingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT casting_id FROM saved_castings WHERE user_id = $1 ORDER BY created_at DESC`

	var ids []uuid.UUID
	err := r.db.SelectContext(ctx, &ids, query, userID)
	return ids, err
}

// GetSavedCastings returns paginated saved castings for a user
func (r *SavedCastingRepository) GetSavedCastings(ctx context.Context, userID uuid.UUID, limit, offset int) ([]SavedCasting, error) {
	query := `
		SELECT id, user_id, casting_id, created_at
		FROM saved_castings
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var saved []SavedCasting
	err := r.db.SelectContext(ctx, &saved, query, userID, limit, offset)
	return saved, err
}

// CountSaved returns total saved castings for a user
func (r *SavedCastingRepository) CountSaved(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM saved_castings WHERE user_id = $1`

	var count int
	err := r.db.GetContext(ctx, &count, query, userID)
	return count, err
}

// GetSavedMap returns a map of castingID -> isSaved for a list of casting IDs
func (r *SavedCastingRepository) GetSavedMap(ctx context.Context, userID uuid.UUID, castingIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	if len(castingIDs) == 0 {
		return make(map[uuid.UUID]bool), nil
	}

	query, args, err := sqlx.In(`
		SELECT casting_id FROM saved_castings 
		WHERE user_id = ? AND casting_id IN (?)
	`, userID, castingIDs)
	if err != nil {
		return nil, err
	}

	query = r.db.Rebind(query)

	var savedIDs []uuid.UUID
	if err := r.db.SelectContext(ctx, &savedIDs, query, args...); err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]bool)
	for _, id := range savedIDs {
		result[id] = true
	}
	return result, nil
}
