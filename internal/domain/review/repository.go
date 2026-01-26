package review

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository handles review database operations
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new review repository
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new review
func (r *Repository) Create(ctx context.Context, review *Review) error {
	query := `
		INSERT INTO reviews (id, profile_id, reviewer_id, casting_id, rating, comment, is_public, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		review.ID,
		review.ProfileID,
		review.ReviewerID,
		review.CastingID,
		review.Rating,
		review.Comment,
		review.IsPublic,
		review.CreatedAt,
		review.UpdatedAt,
	)
	return err
}

// GetByID returns a review by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Review, error) {
	query := `SELECT * FROM reviews WHERE id = $1`
	var review Review
	err := r.db.GetContext(ctx, &review, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &review, err
}

// GetByProfileID returns reviews for a profile
func (r *Repository) GetByProfileID(ctx context.Context, profileID uuid.UUID, limit, offset int) ([]Review, error) {
	query := `
		SELECT * FROM reviews 
		WHERE profile_id = $1 AND is_public = true
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	var reviews []Review
	err := r.db.SelectContext(ctx, &reviews, query, profileID, limit, offset)
	return reviews, err
}

// CountByProfileID returns total reviews for a profile
func (r *Repository) CountByProfileID(ctx context.Context, profileID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM reviews WHERE profile_id = $1 AND is_public = true`
	var count int
	err := r.db.GetContext(ctx, &count, query, profileID)
	return count, err
}

// GetAverageRating returns average rating for a profile
func (r *Repository) GetAverageRating(ctx context.Context, profileID uuid.UUID) (float64, error) {
	query := `SELECT COALESCE(AVG(rating), 0) FROM reviews WHERE profile_id = $1 AND is_public = true`
	var avg float64
	err := r.db.GetContext(ctx, &avg, query, profileID)
	return avg, err
}

// GetRatingDistribution returns count of each rating for a profile
func (r *Repository) GetRatingDistribution(ctx context.Context, profileID uuid.UUID) (map[int]int, error) {
	query := `
		SELECT rating, COUNT(*) as count
		FROM reviews
		WHERE profile_id = $1 AND is_public = true
		GROUP BY rating
	`
	type RatingCount struct {
		Rating int `db:"rating"`
		Count  int `db:"count"`
	}
	var counts []RatingCount
	err := r.db.SelectContext(ctx, &counts, query, profileID)
	if err != nil {
		return nil, err
	}

	dist := make(map[int]int)
	for i := 1; i <= 5; i++ {
		dist[i] = 0
	}
	for _, c := range counts {
		dist[c.Rating] = c.Count
	}
	return dist, nil
}

// Delete removes a review
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM reviews WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// HasReviewed checks if user already reviewed a profile for a casting
func (r *Repository) HasReviewed(ctx context.Context, profileID, reviewerID uuid.UUID, castingID uuid.NullUUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM reviews 
			WHERE profile_id = $1 AND reviewer_id = $2 AND casting_id IS NOT DISTINCT FROM $3
		)
	`
	var exists bool
	err := r.db.GetContext(ctx, &exists, query, profileID, reviewerID, castingID)
	return exists, err
}
