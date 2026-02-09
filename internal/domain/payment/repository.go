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
	// RoboKassa methods (new)
	GetByInvoiceID(ctx context.Context, invoiceID int64) (*Payment, error)
	GetNextInvoiceID(ctx context.Context) (int64, error)
	UpdateByInvoiceID(ctx context.Context, invoiceID int64, status Status) error
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

// GetByInvoiceID retrieves payment by RoboKassa invoice ID
func (r *repository) GetByInvoiceID(ctx context.Context, invoiceID int64) (*Payment, error) {
	query := `
		SELECT id, user_id, plan_id, subscription_id, amount, currency, status, provider, 
		       external_id, description, metadata, robokassa_inv_id, kaspi_order_id,
		       paid_at, failed_at, refunded_at, created_at, updated_at, promotion_id
		FROM payments
		WHERE robokassa_inv_id = $1
		LIMIT 1
	`

	var p Payment
	var roboInvID sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, invoiceID).Scan(
		&p.ID, &p.UserID, &p.PlanID, &p.SubscriptionID, &p.Amount, &p.Currency, &p.Status, &p.Provider,
		&p.ExternalID, &p.Description, &p.Metadata, &roboInvID, &p.KaspiOrderID,
		&p.PaidAt, &p.FailedAt, &p.RefundedAt, &p.CreatedAt, &p.UpdatedAt, &p.PromotionID,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get payment by invoice_id: %w", err)
	}

	if roboInvID.Valid {
		p.RoboKassaInvID = &roboInvID.Int64
	}

	return &p, nil
}

// GetNextInvoiceID generates next available invoice ID for RoboKassa using database sequence
func (r *repository) GetNextInvoiceID(ctx context.Context) (int64, error) {
	query := `SELECT nextval('robokassa_invoice_seq')`

	var invoiceID int64
	err := r.db.QueryRowContext(ctx, query).Scan(&invoiceID)
	if err != nil {
		// Fallback: get next ID from max existing ID
		return r.getNextInvoiceIDFallback(ctx)
	}

	return invoiceID, nil
}

// getNextInvoiceIDFallback generates invoice ID if sequence doesn't exist
func (r *repository) getNextInvoiceIDFallback(ctx context.Context) (int64, error) {
	query := `
		SELECT COALESCE(MAX(robokassa_inv_id), 0) + 1
		FROM payments
		WHERE robokassa_inv_id IS NOT NULL
	`

	var invoiceID int64
	err := r.db.QueryRowContext(ctx, query).Scan(&invoiceID)
	if err != nil {
		return 0, fmt.Errorf("failed to generate invoice_id: %w", err)
	}

	// Ensure minimum starting value
	if invoiceID < 1000 {
		invoiceID = 1000
	}

	return invoiceID, nil
}

// UpdateByInvoiceID updates payment status by invoice ID
func (r *repository) UpdateByInvoiceID(ctx context.Context, invoiceID int64, status Status) error {
	var query string
	switch status {
	case StatusCompleted:
		query = `UPDATE payments SET status = $2, paid_at = NOW(), updated_at = NOW() WHERE robokassa_inv_id = $1`
	case StatusFailed:
		query = `UPDATE payments SET status = $2, failed_at = NOW(), updated_at = NOW() WHERE robokassa_inv_id = $1`
	case StatusRefunded:
		query = `UPDATE payments SET status = $2, refunded_at = NOW(), updated_at = NOW() WHERE robokassa_inv_id = $1`
	default:
		query = `UPDATE payments SET status = $2, updated_at = NOW() WHERE robokassa_inv_id = $1`
	}

	result, err := r.db.ExecContext(ctx, query, invoiceID, status)
	if err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("no payment found with invoice_id %d", invoiceID)
	}

	return nil
}
