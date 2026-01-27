package payment

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines payment data access
type Repository interface {
	Create(ctx context.Context, p *Payment) error
	GetByID(ctx context.Context, id uuid.UUID) (*Payment, error)
	GetByExternalID(ctx context.Context, provider, externalID string) (*Payment, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Payment, error)
}

type repository struct {
	db *sqlx.DB
}

// NewRepository creates payment repository
func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, p *Payment) error {
	query := `
		INSERT INTO payments (id, user_id, subscription_id, amount, currency, status, provider, external_id, description, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.ExecContext(ctx, query,
		p.ID,
		p.UserID,
		p.SubscriptionID,
		p.Amount,
		p.Currency,
		p.Status,
		p.Provider,
		p.ExternalID,
		p.Description,
		p.Metadata,
		p.CreatedAt,
	)
	return err
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Payment, error) {
	query := `SELECT * FROM payments WHERE id = $1`
	var p Payment
	err := r.db.GetContext(ctx, &p, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *repository) GetByExternalID(ctx context.Context, provider, externalID string) (*Payment, error) {
	query := `SELECT * FROM payments WHERE provider = $1 AND external_id = $2`
	var p Payment
	err := r.db.GetContext(ctx, &p, query, provider, externalID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *repository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error {
	var query string
	switch status {
	case StatusCompleted:
		query = `UPDATE payments SET status = $2, paid_at = NOW() WHERE id = $1`
	case StatusFailed:
		query = `UPDATE payments SET status = $2, failed_at = NOW() WHERE id = $1`
	case StatusRefunded:
		query = `UPDATE payments SET status = $2, refunded_at = NOW() WHERE id = $1`
	default:
		query = `UPDATE payments SET status = $2 WHERE id = $1`
	}
	_, err := r.db.ExecContext(ctx, query, id, status)
	return err
}

func (r *repository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Payment, error) {
	query := `
		SELECT * FROM payments 
		WHERE user_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2 OFFSET $3
	`
	var payments []*Payment
	err := r.db.SelectContext(ctx, &payments, query, userID, limit, offset)
	return payments, err
}
