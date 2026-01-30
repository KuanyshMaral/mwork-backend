package casting

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Filter represents search filters
type Filter struct {
	Query     *string
	City      *string
	PayMin    *float64
	PayMax    *float64
	Status    *Status
	CreatorID *uuid.UUID
}

// SortBy represents sort options
type SortBy string

const (
	SortByNewest  SortBy = "newest"
	SortByPayDesc SortBy = "pay_desc"
	SortByPopular SortBy = "popular"
)

// Pagination for listing
type Pagination struct {
	Page  int
	Limit int
}

// Repository defines casting data access interface
type Repository interface {
	Create(ctx context.Context, casting *Casting) error
	GetByID(ctx context.Context, id uuid.UUID) (*Casting, error)
	Update(ctx context.Context, casting *Casting) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter *Filter, sortBy SortBy, pagination *Pagination) ([]*Casting, int, error)
	IncrementViewCount(ctx context.Context, id uuid.UUID) error
	ListByCreator(ctx context.Context, creatorID uuid.UUID, pagination *Pagination) ([]*Casting, int, error)
	CountActiveByCreatorID(ctx context.Context, creatorID string) (int, error)
}

type repository struct {
	db *sqlx.DB
}

// NewRepository creates new casting repository
func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, casting *Casting) error {
	query := `
		INSERT INTO castings (
			id, creator_id, title, description, city, address,
			pay_min, pay_max, pay_type, date_from, date_to,
			cover_image_url,
			requirements, status, is_promoted, view_count, response_count
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12,
			$13, $14, $15, $16, $17
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		casting.ID, casting.CreatorID, casting.Title, casting.Description, casting.City, casting.Address,
		casting.PayMin, casting.PayMax, casting.PayType, casting.DateFrom, casting.DateTo,
		casting.CoverImageURL,
		casting.Requirements, casting.Status, casting.IsPromoted, casting.ViewCount, casting.ResponseCount,
	)

	return err
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Casting, error) {
	query := `
		SELECT * FROM castings
		WHERE id = $1 AND status != 'deleted'
	`

	var casting Casting
	err := r.db.GetContext(ctx, &casting, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &casting, nil
}

func (r *repository) Update(ctx context.Context, casting *Casting) error {
	query := `
		UPDATE castings SET
			title = $2, description = $3, city = $4, address = $5,
			pay_min = $6, pay_max = $7, pay_type = $8,
			date_from = $9, date_to = $10,
			cover_image_url = $11,
			requirements = $12, status = $13,
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		casting.ID,
		casting.Title, casting.Description, casting.City, casting.Address,
		casting.PayMin, casting.PayMax, casting.PayType,
		casting.DateFrom, casting.DateTo,
		casting.Requirements, casting.Status,
	)

	return err
}

func (r *repository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error {
	query := `UPDATE castings SET status = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, status)
	return err
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE castings SET status = 'deleted', updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *repository) List(ctx context.Context, filter *Filter, sortBy SortBy, pagination *Pagination) ([]*Casting, int, error) {
	conditions := []string{"c.status != 'deleted'"}
	args := []interface{}{}
	argIndex := 1

	// Default to active status if not specified
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("c.status = $%d", argIndex))
		args = append(args, *filter.Status)
		argIndex++
	} else {
		conditions = append(conditions, "c.status = 'active'")
	}

	if filter.City != nil && *filter.City != "" {
		conditions = append(conditions, fmt.Sprintf("c.city ILIKE $%d", argIndex))
		args = append(args, "%"+*filter.City+"%")
		argIndex++
	}

	if filter.PayMin != nil {
		conditions = append(conditions, fmt.Sprintf("(c.pay_max >= $%d OR c.pay_max IS NULL)", argIndex))
		args = append(args, *filter.PayMin)
		argIndex++
	}

	if filter.PayMax != nil {
		conditions = append(conditions, fmt.Sprintf("(c.pay_min <= $%d OR c.pay_min IS NULL)", argIndex))
		args = append(args, *filter.PayMax)
		argIndex++
	}

	if filter.CreatorID != nil {
		conditions = append(conditions, fmt.Sprintf("c.creator_id = $%d", argIndex))
		args = append(args, *filter.CreatorID)
		argIndex++
	}

	if filter.Query != nil && *filter.Query != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(c.title ILIKE $%d OR c.description ILIKE $%d)",
			argIndex, argIndex,
		))
		args = append(args, "%"+*filter.Query+"%")
		argIndex++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM castings c %s", where)
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	// Order by
	var orderBy string
	switch sortBy {
	case SortByPayDesc:
		orderBy = "ORDER BY c.pay_max DESC NULLS LAST"
	case SortByPopular:
		orderBy = "ORDER BY c.view_count DESC, c.response_count DESC"
	default:
		orderBy = "ORDER BY c.created_at DESC"
	}

	// Get castings with pagination
	offset := (pagination.Page - 1) * pagination.Limit
	query := fmt.Sprintf(`
		SELECT * FROM castings c
		%s %s
		LIMIT $%d OFFSET $%d
	`, where, orderBy, argIndex, argIndex+1)
	args = append(args, pagination.Limit, offset)

	var castings []*Casting
	if err := r.db.SelectContext(ctx, &castings, query, args...); err != nil {
		return nil, 0, err
	}

	return castings, total, nil
}

func (r *repository) IncrementViewCount(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE castings SET view_count = view_count + 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *repository) ListByCreator(ctx context.Context, creatorID uuid.UUID, pagination *Pagination) ([]*Casting, int, error) {
	// Count
	countQuery := `SELECT COUNT(*) FROM castings WHERE creator_id = $1 AND status != 'deleted'`
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, creatorID); err != nil {
		return nil, 0, err
	}

	// List
	offset := (pagination.Page - 1) * pagination.Limit
	query := `
		SELECT * FROM castings 
		WHERE creator_id = $1 AND status != 'deleted'
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var castings []*Casting
	if err := r.db.SelectContext(ctx, &castings, query, creatorID, pagination.Limit, offset); err != nil {
		return nil, 0, err
	}

	return castings, total, nil
}

func (r *repository) CountActiveByCreatorID(ctx context.Context, creatorID string) (int, error) {
	query := `
        SELECT COUNT(*) 
        FROM castings 
        WHERE creator_id = $1 AND status IN ('active', 'draft')`
	var count int
	err := r.db.QueryRowContext(ctx, query, creatorID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count active castings: %w", err)
	}
	return count, nil
}
