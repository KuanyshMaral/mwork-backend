package favorite

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// EntityType defines what can be favorited
type EntityType string

const (
	EntityTypeCasting EntityType = "casting"
	EntityTypeProfile EntityType = "profile"
)

// Favorite represents a bookmarked item
type Favorite struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	EntityType EntityType `json:"entity_type" db:"entity_type"`
	EntityID   uuid.UUID  `json:"entity_id" db:"entity_id"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// Repository for favorites
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates favorites repository
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// Add adds a favorite
func (r *Repository) Add(ctx context.Context, userID uuid.UUID, entityType EntityType, entityID uuid.UUID) (*Favorite, error) {
	fav := &Favorite{
		ID:         uuid.New(),
		UserID:     userID,
		EntityType: entityType,
		EntityID:   entityID,
		CreatedAt:  time.Now(),
	}

	query := `
		INSERT INTO favorites (id, user_id, entity_type, entity_id, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, entity_type, entity_id) DO NOTHING
		RETURNING id
	`

	var insertedID uuid.UUID
	err := r.db.GetContext(ctx, &insertedID, query,
		fav.ID, fav.UserID, fav.EntityType, fav.EntityID, fav.CreatedAt)
	if err != nil {
		// Check if already exists
		existing, _ := r.GetByUserAndEntity(ctx, userID, entityType, entityID)
		if existing != nil {
			return existing, nil
		}
		return nil, err
	}

	return fav, nil
}

// Remove removes a favorite
func (r *Repository) Remove(ctx context.Context, userID uuid.UUID, entityType EntityType, entityID uuid.UUID) error {
	query := `DELETE FROM favorites WHERE user_id = $1 AND entity_type = $2 AND entity_id = $3`
	_, err := r.db.ExecContext(ctx, query, userID, entityType, entityID)
	return err
}

// GetByUserAndEntity checks if entity is favorited
func (r *Repository) GetByUserAndEntity(ctx context.Context, userID uuid.UUID, entityType EntityType, entityID uuid.UUID) (*Favorite, error) {
	var fav Favorite
	query := `SELECT * FROM favorites WHERE user_id = $1 AND entity_type = $2 AND entity_id = $3`
	err := r.db.GetContext(ctx, &fav, query, userID, entityType, entityID)
	if err != nil {
		return nil, err
	}
	return &fav, nil
}

// IsFavorited checks if entity is favorited by user
func (r *Repository) IsFavorited(ctx context.Context, userID uuid.UUID, entityType EntityType, entityID uuid.UUID) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM favorites WHERE user_id = $1 AND entity_type = $2 AND entity_id = $3`
	err := r.db.GetContext(ctx, &count, query, userID, entityType, entityID)
	return count > 0, err
}

// ListByUser returns all favorites for a user
func (r *Repository) ListByUser(ctx context.Context, userID uuid.UUID, entityType *EntityType, limit, offset int) ([]*Favorite, int, error) {
	args := []interface{}{userID}
	query := `SELECT * FROM favorites WHERE user_id = $1`
	countQuery := `SELECT COUNT(*) FROM favorites WHERE user_id = $1`

	if entityType != nil {
		query += ` AND entity_type = $2`
		countQuery += ` AND entity_type = $2`
		args = append(args, *entityType)
	}

	query += ` ORDER BY created_at DESC LIMIT $` + string(rune('0'+len(args)+1)) + ` OFFSET $` + string(rune('0'+len(args)+2))

	// Count total
	var total int
	if entityType != nil {
		r.db.GetContext(ctx, &total, countQuery, userID, *entityType)
	} else {
		r.db.GetContext(ctx, &total, countQuery, userID)
	}

	// Add limit/offset
	args = append(args, limit, offset)

	// Simplified query execution
	var favs []*Favorite
	if entityType != nil {
		err := r.db.SelectContext(ctx, &favs,
			`SELECT * FROM favorites WHERE user_id = $1 AND entity_type = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
			userID, *entityType, limit, offset)
		return favs, total, err
	}

	err := r.db.SelectContext(ctx, &favs,
		`SELECT * FROM favorites WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	return favs, total, err
}

// CountByEntity returns how many users favorited an entity
func (r *Repository) CountByEntity(ctx context.Context, entityType EntityType, entityID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM favorites WHERE entity_type = $1 AND entity_id = $2`
	err := r.db.GetContext(ctx, &count, query, entityType, entityID)
	return count, err
}
