package casting

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"

	"github.com/mwork/mwork-api/internal/middleware"
)

// Filter represents search filters
type Filter struct {
	Query     *string
	City      *string
	PayMin    *float64
	PayMax    *float64
	Status    *Status
	CreatorID *uuid.UUID
	IsUrgent  *bool
	WorkType  *string
	Tags      []string
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
	IncrementAcceptedAndMaybeClose(ctx context.Context, id uuid.UUID) (int, Status, error)
	IncrementAcceptedAndMaybeCloseTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID) (int, Status, error)
	IncrementResponseCount(ctx context.Context, id uuid.UUID, delta int) error
	IncrementResponseCountTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, delta int) error
	ListByCreator(ctx context.Context, creatorID uuid.UUID, pagination *Pagination) ([]*Casting, int, error)
	CountActiveByCreatorID(ctx context.Context, creatorID string) (int, error)
}

type repository struct {
	db *sqlx.DB
}

const castingSelectColumns = `
	id, creator_id, title, description, city, address,
	pay_min, pay_max, pay_type, date_from, date_to,
	cover_image_url,
	required_gender, min_age, max_age, min_height, max_height,
	min_weight, max_weight, required_experience, required_languages,
	clothing_sizes, shoe_sizes,
	work_type, event_datetime, event_location, deadline_at, is_urgent,
	status, is_promoted, view_count, response_count,
	created_at, updated_at, moderation_status, required_models_count,
	accepted_models_count, tags, rating_score, reviews_count
`

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
			required_gender, min_age, max_age, min_height, max_height,
			min_weight, max_weight, required_experience, required_languages,
			clothing_sizes, shoe_sizes,
			work_type, event_datetime, event_location, deadline_at, is_urgent,
			status, is_promoted, view_count, response_count,
			tags
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12,
			$13, $14, $15, $16, $17,
			$18, $19, $20, $21,
			$22, $23,
			$24, $25, $26, $27, $28,
			$29, $30, $31, $32,
			$33
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		casting.ID, casting.CreatorID, casting.Title, casting.Description, casting.City, casting.Address,
		casting.PayMin, casting.PayMax, casting.PayType, casting.DateFrom, casting.DateTo,
		casting.CoverImageURL,
		casting.RequiredGender, casting.AgeMin, casting.AgeMax, casting.HeightMin, casting.HeightMax,
		casting.WeightMin, casting.WeightMax, casting.RequiredExperience, casting.RequiredLanguages,
		casting.ClothingSizes, casting.ShoeSizes,
		casting.WorkType, casting.EventDatetime, casting.EventLocation, casting.DeadlineAt, casting.IsUrgent,
		casting.Status, casting.IsPromoted, casting.ViewCount, casting.ResponseCount,
		casting.Tags,
	)
	if err != nil {
		evt := log.Error().
			Str("request_id", middleware.GetRequestID(ctx)).
			Str("query", "castings.create").
			Str("casting_id", casting.ID.String()).
			Str("creator_id", casting.CreatorID.String()).
			Str("status", string(casting.Status)).
			Str("pay_type", casting.PayType).
			Interface("date_from", casting.DateFrom).
			Interface("date_to", casting.DateTo).
			Err(err)

		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			evt = evt.
				Str("pg_code", string(pqErr.Code)).
				Str("pg_constraint", pqErr.Constraint)
		}

		evt.Msg("casting insert failed")
		return mapCreateDBError(err)
	}

	return nil
}

func mapCreateDBError(err error) error {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return err
	}

	constraint := strings.ToLower(pqErr.Constraint)
	switch pqErr.Code {
	case "23514":
		switch {
		case constraint == "valid_pay_range" || strings.Contains(constraint, "pay_range"):
			return fmt.Errorf("%w: %w", ErrInvalidPayRange, err)
		case constraint == "valid_date_range" || strings.Contains(constraint, "date_range"):
			return fmt.Errorf("%w: %w", ErrInvalidDateRange, err)
		default:
			return fmt.Errorf("%w: %w", ErrCastingConstraint, err)
		}
	case "23503":
		return fmt.Errorf("%w: %w", ErrInvalidCreatorReference, err)
	case "23505":
		return fmt.Errorf("%w: %w", ErrDuplicateCasting, err)
	default:
		return err
	}
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Casting, error) {
	query := `
		SELECT ` + castingSelectColumns + ` FROM castings
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
			required_gender = $12, min_age = $13, max_age = $14,
			min_height = $15, max_height = $16,
			min_weight = $17, max_weight = $18,
			required_experience = $19, required_languages = $20,
			clothing_sizes = $21, shoe_sizes = $22,
			work_type = $23, event_datetime = $24, event_location = $25,
			deadline_at = $26, is_urgent = $27,
			status = $28,
			tags = $29,
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		casting.ID,
		casting.Title, casting.Description, casting.City, casting.Address,
		casting.PayMin, casting.PayMax, casting.PayType,
		casting.DateFrom, casting.DateTo,
		casting.CoverImageURL,
		casting.RequiredGender, casting.AgeMin, casting.AgeMax,
		casting.HeightMin, casting.HeightMax,
		casting.WeightMin, casting.WeightMax,
		casting.RequiredExperience, casting.RequiredLanguages,
		casting.ClothingSizes, casting.ShoeSizes,
		casting.WorkType, casting.EventDatetime, casting.EventLocation,
		casting.DeadlineAt, casting.IsUrgent,
		casting.Status,
		casting.Tags,
	)
	if err != nil {
		return mapCreateDBError(err)
	}

	return nil
}

func (r *repository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error {
	query := `UPDATE castings SET status = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, status)
	if err != nil {
		return mapCreateDBError(err)
	}
	return nil
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

	if filter.IsUrgent != nil {
		conditions = append(conditions, fmt.Sprintf("c.is_urgent = $%d", argIndex))
		args = append(args, *filter.IsUrgent)
		argIndex++
	}

	if filter.WorkType != nil && *filter.WorkType != "" {
		conditions = append(conditions, fmt.Sprintf("c.work_type = $%d", argIndex))
		args = append(args, *filter.WorkType)
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

	if len(filter.Tags) > 0 {
		conditions = append(conditions, fmt.Sprintf("c.tags @> $%d", argIndex))
		args = append(args, pq.StringArray(filter.Tags))
		argIndex++
	}

	if len(filter.Tags) > 0 {
		conditions = append(conditions, fmt.Sprintf("c.tags @> $%d", argIndex))
		args = append(args, pq.StringArray(filter.Tags))
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
		orderBy = "ORDER BY c.is_urgent DESC, c.created_at DESC"
	}

	// Get castings with pagination
	offset := (pagination.Page - 1) * pagination.Limit
	query := fmt.Sprintf(`
		SELECT %s FROM castings c
		%s %s
		LIMIT $%d OFFSET $%d
	`, castingSelectColumns, where, orderBy, argIndex, argIndex+1)
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

func (r *repository) IncrementAcceptedAndMaybeClose(ctx context.Context, id uuid.UUID) (int, Status, error) {
	return r.incrementAcceptedAndMaybeClose(ctx, r.db, id)
}

func (r *repository) IncrementAcceptedAndMaybeCloseTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID) (int, Status, error) {
	return r.incrementAcceptedAndMaybeClose(ctx, tx, id)
}

func (r *repository) incrementAcceptedAndMaybeClose(ctx context.Context, execer sqlx.ExtContext, id uuid.UUID) (int, Status, error) {
	query := `
		UPDATE castings
		SET accepted_models_count = accepted_models_count + 1,
			status = CASE
				WHEN required_models_count IS NOT NULL
					AND accepted_models_count + 1 >= required_models_count
				THEN 'closed'
				ELSE status
			END
		WHERE id = $1
		  AND status != 'closed'
		  AND (required_models_count IS NULL OR accepted_models_count < required_models_count)
		RETURNING accepted_models_count, status
	`

	var accepted int
	var status Status
	err := execer.QueryRowxContext(ctx, query, id).Scan(&accepted, &status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", ErrCastingFullOrClosed
		}
		return 0, "", err
	}

	return accepted, status, nil
}

func (r *repository) IncrementResponseCount(ctx context.Context, id uuid.UUID, delta int) error {
	return r.incrementResponseCount(ctx, r.db, id, delta)
}

func (r *repository) IncrementResponseCountTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, delta int) error {
	return r.incrementResponseCount(ctx, tx, id, delta)
}

func (r *repository) incrementResponseCount(ctx context.Context, execer sqlx.ExtContext, id uuid.UUID, delta int) error {
	query := `UPDATE castings SET response_count = response_count + $2 WHERE id = $1`
	_, err := execer.ExecContext(ctx, query, id, delta)
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
		SELECT ` + castingSelectColumns + ` FROM castings 
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
