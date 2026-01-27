package response

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Pagination for listing
type Pagination struct {
	Page  int
	Limit int
}

// Repository defines response data access interface
type Repository interface {
	Create(ctx context.Context, response *Response) error
	GetByID(ctx context.Context, id uuid.UUID) (*Response, error)
	GetByModelAndCasting(ctx context.Context, modelID, castingID uuid.UUID) (*Response, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByCasting(ctx context.Context, castingID uuid.UUID, pagination *Pagination) ([]*Response, int, error)
	ListByModel(ctx context.Context, modelID uuid.UUID, pagination *Pagination) ([]*Response, int, error)
}

type repository struct {
	db *sqlx.DB
}

// NewRepository creates new response repository
func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, response *Response) error {
	query := `
		INSERT INTO casting_responses (id, casting_id, model_id, message, proposed_rate, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		response.ID,
		response.CastingID,
		response.ModelID,
		response.Message,
		response.ProposedRate,
		response.Status,
	)

	return err
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Response, error) {
	query := `
		SELECT cr.*, c.title as casting_title, c.city as casting_city, mp.name as model_name
		FROM casting_responses cr
		LEFT JOIN castings c ON cr.casting_id = c.id
		LEFT JOIN model_profiles mp ON cr.model_id = mp.id
		WHERE cr.id = $1
	`

	var response Response
	err := r.db.GetContext(ctx, &response, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &response, nil
}

func (r *repository) GetByModelAndCasting(ctx context.Context, modelID, castingID uuid.UUID) (*Response, error) {
	query := `SELECT * FROM casting_responses WHERE model_id = $1 AND casting_id = $2`

	var response Response
	err := r.db.GetContext(ctx, &response, query, modelID, castingID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &response, nil
}

func (r *repository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error {
	var query string
	switch status {
	case StatusAccepted:
		query = `UPDATE casting_responses SET status = $2, accepted_at = NOW(), updated_at = NOW() WHERE id = $1`
	case StatusRejected:
		query = `UPDATE casting_responses SET status = $2, rejected_at = NOW(), updated_at = NOW() WHERE id = $1`
	default:
		query = `UPDATE casting_responses SET status = $2, updated_at = NOW() WHERE id = $1`
	}
	_, err := r.db.ExecContext(ctx, query, id, status)
	return err
}

func (r *repository) ListByCasting(ctx context.Context, castingID uuid.UUID, pagination *Pagination) ([]*Response, int, error) {
	// Count
	countQuery := `SELECT COUNT(*) FROM casting_responses WHERE casting_id = $1`
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, castingID); err != nil {
		return nil, 0, err
	}

	// Get responses with model info
	offset := (pagination.Page - 1) * pagination.Limit
	query := `
		SELECT cr.*, mp.name as model_name
		FROM casting_responses cr
		LEFT JOIN model_profiles mp ON cr.model_id = mp.id
		WHERE cr.casting_id = $1 
		ORDER BY cr.created_at DESC 
		LIMIT $2 OFFSET $3
	`

	var responses []*Response
	if err := r.db.SelectContext(ctx, &responses, query, castingID, pagination.Limit, offset); err != nil {
		return nil, 0, err
	}

	return responses, total, nil
}

func (r *repository) ListByModel(ctx context.Context, modelID uuid.UUID, pagination *Pagination) ([]*Response, int, error) {
	// Count
	countQuery := `SELECT COUNT(*) FROM casting_responses WHERE model_id = $1`
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, modelID); err != nil {
		return nil, 0, err
	}

	// Get responses with casting info
	offset := (pagination.Page - 1) * pagination.Limit
	query := `
		SELECT cr.*, c.title as casting_title, c.city as casting_city
		FROM casting_responses cr
		LEFT JOIN castings c ON cr.casting_id = c.id
		WHERE cr.model_id = $1 
		ORDER BY cr.created_at DESC 
		LIMIT $2 OFFSET $3
	`

	var responses []*Response
	if err := r.db.SelectContext(ctx, &responses, query, modelID, pagination.Limit, offset); err != nil {
		return nil, 0, err
	}

	return responses, total, nil
}

// Delete removes a response (hard delete)
func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM casting_responses WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
