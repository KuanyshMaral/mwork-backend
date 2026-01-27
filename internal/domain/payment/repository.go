package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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
	CreatePendingPayment(ctx context.Context, payment *Payment) error
	ConfirmPayment(ctx context.Context, kaspiOrderID string) error
	GetByPromotionID(ctx context.Context, promotionID string) (*Payment, error)
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

func (r *repository) CreatePendingPayment(ctx context.Context, payment *Payment) error {
	query := `
        INSERT INTO payments (user_id, plan_id, amount, kaspi_order_id, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, 'pending', NOW(), NOW())
        RETURNING id`
	err := r.db.QueryRowContext(ctx, query, payment.UserID, payment.PlanID, payment.Amount, payment.KaspiOrderID).Scan(&payment.ID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return fmt.Errorf("payment already exists: %w", err)
		}
		return fmt.Errorf("database error: %w", err)
	}
	return nil
}

func (r *repository) ConfirmPayment(ctx context.Context, kaspiOrderID string) error {
	query := `
        UPDATE payments
        SET status = 'completed', updated_at = NOW()
        WHERE kaspi_order_id = $1 AND status = 'pending'`
	result, err := r.db.ExecContext(ctx, query, kaspiOrderID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *repository) GetByPromotionID(ctx context.Context, promotionID string) (*Payment, error) {
	query := `
        SELECT id, user_id, plan_id, amount, kaspi_order_id, status, created_at, updated_at, promotion_id
        FROM payments
        WHERE promotion_id = $1
        ORDER BY created_at DESC
        LIMIT 1`
	var p Payment
	err := r.db.QueryRowContext(ctx, query, promotionID).Scan(&p.ID, &p.UserID, &p.PlanID, &p.Amount, &p.KaspiOrderID, &p.Status, &p.CreatedAt, &p.UpdatedAt, &p.PromotionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, fmt.Errorf("failed to scan payment: %w", err)
	}
	return &p, nil
}
