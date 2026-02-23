package review

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository handles review database operations.
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new review repository.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new review inside a transaction and updates cached stats on the target.
func (r *Repository) Create(ctx context.Context, review *Review) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	criteriaJSON, err := json.Marshal(review.Criteria)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO reviews
			(id, author_id, target_type, target_id, context_type, context_id,
			 rating, comment, criteria, is_public, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err = tx.ExecContext(ctx, query,
		review.ID, review.AuthorID, review.TargetType, review.TargetID,
		review.ContextType, review.ContextID,
		review.Rating, review.Comment, criteriaJSON,
		review.IsPublic, review.CreatedAt, review.UpdatedAt,
	)
	if err != nil {
		return err
	}

	if err = r.updateCachedStats(ctx, tx, review.TargetType, review.TargetID); err != nil {
		return err
	}

	return tx.Commit()
}

// Delete removes a review and updates cached stats.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	// Fetch target info before deleting
	var rev Review
	if err := r.db.GetContext(ctx, &rev, `SELECT target_type, target_id FROM reviews WHERE id = $1`, id); err != nil {
		return err
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, `DELETE FROM reviews WHERE id = $1`, id); err != nil {
		return err
	}

	if err = r.updateCachedStats(ctx, tx, rev.TargetType, rev.TargetID); err != nil {
		return err
	}

	return tx.Commit()
}

// updateCachedStats recalculates rating_score and reviews_count for a target entity.
func (r *Repository) updateCachedStats(ctx context.Context, tx *sqlx.Tx, targetType TargetType, targetID uuid.UUID) error {
	var avg float64
	var count int

	err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(AVG(rating), 0), COUNT(*) FROM reviews WHERE target_type = $1 AND target_id = $2 AND is_public = true`,
		targetType, targetID,
	).Scan(&avg, &count)
	if err != nil {
		return err
	}

	var updateSQL string
	switch targetType {
	case TargetTypeModelProfile:
		updateSQL = `UPDATE model_profiles SET rating_score = $1, reviews_count = $2 WHERE id = $3`
	case TargetTypeEmployerProfile:
		updateSQL = `UPDATE employer_profiles SET rating_score = $1, reviews_count = $2 WHERE id = $3`
	case TargetTypeCasting:
		updateSQL = `UPDATE castings SET rating_score = $1, reviews_count = $2 WHERE id = $3`
	default:
		return fmt.Errorf("unsupported target_type: %s", targetType)
	}

	_, err = tx.ExecContext(ctx, updateSQL, avg, count, targetID)
	return err
}

// GetByID returns a review by ID.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Review, error) {
	var review Review
	query := `
		SELECT r.*, u.full_name as author_name
		FROM reviews r
		LEFT JOIN users u ON r.author_id = u.id
		WHERE r.id = $1
	`
	err := r.db.GetContext(ctx, &review, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &review, err
}

// GetByTarget returns paginated public reviews for a target entity.
func (r *Repository) GetByTarget(ctx context.Context, targetType TargetType, targetID uuid.UUID, limit, offset int) ([]Review, error) {
	query := `
		SELECT r.*, u.full_name as author_name
		FROM reviews r
		LEFT JOIN users u ON r.author_id = u.id
		WHERE r.target_type = $1 AND r.target_id = $2 AND r.is_public = true
		ORDER BY r.created_at DESC
		LIMIT $3 OFFSET $4
	`
	var reviews []Review
	err := r.db.SelectContext(ctx, &reviews, query, targetType, targetID, limit, offset)
	return reviews, err
}

// CountByTarget returns total public reviews for a target entity.
func (r *Repository) CountByTarget(ctx context.Context, targetType TargetType, targetID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM reviews WHERE target_type = $1 AND target_id = $2 AND is_public = true`,
		targetType, targetID,
	)
	return count, err
}

// GetAverageRating returns the average rating for a target entity.
func (r *Repository) GetAverageRating(ctx context.Context, targetType TargetType, targetID uuid.UUID) (float64, error) {
	var avg float64
	err := r.db.GetContext(ctx, &avg,
		`SELECT COALESCE(AVG(rating), 0) FROM reviews WHERE target_type = $1 AND target_id = $2 AND is_public = true`,
		targetType, targetID,
	)
	return avg, err
}

// GetRatingDistribution returns count per star for a target entity.
func (r *Repository) GetRatingDistribution(ctx context.Context, targetType TargetType, targetID uuid.UUID) (map[int]int, error) {
	type row struct {
		Rating int `db:"rating"`
		Count  int `db:"count"`
	}
	var rows []row
	err := r.db.SelectContext(ctx, &rows,
		`SELECT rating, COUNT(*) as count FROM reviews WHERE target_type = $1 AND target_id = $2 AND is_public = true GROUP BY rating`,
		targetType, targetID,
	)
	if err != nil {
		return nil, err
	}
	dist := map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 0}
	for _, row := range rows {
		dist[row.Rating] = row.Count
	}
	return dist, nil
}

// HasReviewed checks if an author already reviewed a specific target+context.
func (r *Repository) HasReviewed(ctx context.Context, authorID uuid.UUID, targetType TargetType, targetID uuid.UUID, contextID uuid.NullUUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `
		SELECT EXISTS(
			SELECT 1 FROM reviews
			WHERE author_id = $1 AND target_type = $2 AND target_id = $3
			  AND context_id IS NOT DISTINCT FROM $4
		)
	`, authorID, targetType, targetID, contextID)
	return exists, err
}
