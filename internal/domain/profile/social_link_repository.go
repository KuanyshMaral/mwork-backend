package profile

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// SocialLinkRepository handles social link database operations
type SocialLinkRepository struct {
	db *sqlx.DB
}

// NewSocialLinkRepository creates a new repository
func NewSocialLinkRepository(db *sqlx.DB) *SocialLinkRepository {
	return &SocialLinkRepository{db: db}
}

// GetByProfileID returns all social links for a profile
func (r *SocialLinkRepository) GetByProfileID(ctx context.Context, profileID uuid.UUID) ([]SocialLink, error) {
	query := `
		SELECT id, profile_id, platform, url, username, is_verified, created_at
		FROM profile_social_links
		WHERE profile_id = $1
		ORDER BY platform
	`

	var links []SocialLink
	err := r.db.SelectContext(ctx, &links, query, profileID)
	if err != nil {
		return nil, err
	}
	return links, nil
}

// Create adds a new social link
func (r *SocialLinkRepository) Create(ctx context.Context, link *SocialLink) error {
	query := `
		INSERT INTO profile_social_links (id, profile_id, platform, url, username, is_verified, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (profile_id, platform) DO UPDATE SET
			url = EXCLUDED.url,
			username = EXCLUDED.username
	`

	_, err := r.db.ExecContext(ctx, query,
		link.ID,
		link.ProfileID,
		link.Platform,
		link.URL,
		link.Username,
		link.IsVerified,
		link.CreatedAt,
	)
	return err
}

// Delete removes a social link
func (r *SocialLinkRepository) Delete(ctx context.Context, profileID uuid.UUID, platform string) error {
	query := `DELETE FROM profile_social_links WHERE profile_id = $1 AND platform = $2`
	result, err := r.db.ExecContext(ctx, query, profileID, platform)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteAll removes all social links for a profile
func (r *SocialLinkRepository) DeleteAll(ctx context.Context, profileID uuid.UUID) error {
	query := `DELETE FROM profile_social_links WHERE profile_id = $1`
	_, err := r.db.ExecContext(ctx, query, profileID)
	return err
}

// UpsertMultiple replaces all social links for a profile
func (r *SocialLinkRepository) UpsertMultiple(ctx context.Context, profileID uuid.UUID, links []SocialLink) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing
	_, err = tx.ExecContext(ctx, `DELETE FROM profile_social_links WHERE profile_id = $1`, profileID)
	if err != nil {
		return err
	}

	// Insert new
	for _, link := range links {
		query := `
			INSERT INTO profile_social_links (id, profile_id, platform, url, username, is_verified, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`
		_, err = tx.ExecContext(ctx, query,
			link.ID,
			link.ProfileID,
			link.Platform,
			link.URL,
			link.Username,
			link.IsVerified,
			link.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
