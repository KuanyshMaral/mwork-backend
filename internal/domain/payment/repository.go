package payment

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

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
	GetByPromotionID(ctx context.Context, promotionID string) (*Payment, error)
	CreateRobokassaPending(ctx context.Context, payment *Payment) error
	GetByRobokassaInvIDForUpdate(ctx context.Context, tx *sqlx.Tx, invID int64) (*Payment, error)
	GetByInvIDForUpdate(ctx context.Context, tx *sqlx.Tx, invID string) (*Payment, error)
	MarkRobokassaSucceeded(ctx context.Context, tx *sqlx.Tx, paymentID uuid.UUID, callbackPayload map[string]string) error
	CreatePaymentEvent(ctx context.Context, tx *sqlx.Tx, paymentID uuid.UUID, eventType string, payload any) error
	BeginTxx(ctx context.Context) (*sqlx.Tx, error)
	NextRobokassaInvID(ctx context.Context) (int64, error)
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
		SELECT 
			id, user_id, plan_id, subscription_id,
			COALESCE(type, '') AS type,
			plan, inv_id,
			response_package, amount, robokassa_inv_id,
			COALESCE(currency, 'KZT') AS currency,
			COALESCE(status, 'pending') AS status,
			provider, external_id, description,
			COALESCE(metadata, 'null'::jsonb) as metadata,
			COALESCE(raw_init_payload, 'null'::jsonb) as raw_init_payload,
			COALESCE(raw_callback_payload, 'null'::jsonb) as raw_callback_payload,
			paid_at, failed_at, refunded_at, created_at,
			COALESCE(updated_at, created_at) AS updated_at,
			promotion_id
		FROM payments 
		WHERE user_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2 OFFSET $3
	`
	var payments []*Payment
	err := r.db.SelectContext(ctx, &payments, query, userID, limit, offset)
	return payments, err
}

func (r *repository) GetByPromotionID(ctx context.Context, promotionID string) (*Payment, error) {
	query := `
        SELECT id, user_id, plan_id, amount, status, created_at, updated_at, promotion_id
        FROM payments
        WHERE promotion_id = $1
        ORDER BY created_at DESC
        LIMIT 1`
	var p Payment
	err := r.db.QueryRowContext(ctx, query, promotionID).Scan(&p.ID, &p.UserID, &p.PlanID, &p.Amount, &p.Status, &p.CreatedAt, &p.UpdatedAt, &p.PromotionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, fmt.Errorf("failed to scan payment: %w", err)
	}
	return &p, nil
}

func (r *repository) CreateRobokassaPending(ctx context.Context, payment *Payment) error {
	query := `
		INSERT INTO payments (id, user_id, subscription_id, type, plan, inv_id, response_package, amount, currency, status, provider, external_id, robokassa_inv_id, description, metadata, raw_init_payload, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, NOW(), NOW())`
	_, err := r.db.ExecContext(ctx, query,
		payment.ID,
		payment.UserID,
		payment.SubscriptionID,
		payment.Type,
		payment.Plan,
		payment.InvID,
		payment.ResponsePackage,
		payment.Amount,
		payment.Currency,
		payment.Status,
		payment.Provider,
		payment.ExternalID,
		payment.RobokassaInvID,
		payment.Description,
		payment.Metadata,
		payment.RawInitPayload,
	)
	if err != nil && isUndefinedPaymentsColumnErr(err) {
		return r.createRobokassaPendingLegacy(ctx, payment)
	}
	return err
}

func (r *repository) createRobokassaPendingLegacy(ctx context.Context, payment *Payment) error {
	query := `
		INSERT INTO payments (id, user_id, subscription_id, amount, currency, status, provider, external_id, description, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())`
	_, err := r.db.ExecContext(ctx, query,
		payment.ID,
		payment.UserID,
		payment.SubscriptionID,
		payment.Amount,
		payment.Currency,
		payment.Status,
		payment.Provider,
		payment.ExternalID,
		payment.Description,
		payment.Metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to create robokassa payment using legacy schema: %w", err)
	}
	return nil
}

func (r *repository) GetByRobokassaInvIDForUpdate(ctx context.Context, tx *sqlx.Tx, invID int64) (*Payment, error) {
	query := `SELECT * FROM payments WHERE provider = 'robokassa' AND robokassa_inv_id = $1 FOR UPDATE`
	var p Payment
	err := tx.GetContext(ctx, &p, query, invID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *repository) GetByInvIDForUpdate(ctx context.Context, tx *sqlx.Tx, invID string) (*Payment, error) {
	query := `SELECT * FROM payments WHERE provider = 'robokassa' AND inv_id = $1 FOR UPDATE`
	var p Payment
	err := tx.GetContext(ctx, &p, query, invID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *repository) MarkRobokassaSucceeded(ctx context.Context, tx *sqlx.Tx, paymentID uuid.UUID, callbackPayload map[string]string) error {
	payloadJSON, err := json.Marshal(callbackPayload)
	if err != nil {
		return err
	}
	query := `
		UPDATE payments
		SET status = 'paid', paid_at = NOW(), updated_at = NOW(), raw_callback_payload = $2
		WHERE id = $1 AND status = 'pending'`
	result, err := tx.ExecContext(ctx, query, paymentID, payloadJSON)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *repository) CreatePaymentEvent(ctx context.Context, tx *sqlx.Tx, paymentID uuid.UUID, eventType string, payload any) error {
	query := `INSERT INTO payment_events (payment_id, event_type, payload, created_at) VALUES ($1, $2, $3, NOW())`
	_, err := tx.ExecContext(ctx, query, paymentID, eventType, payload)
	return err
}

func (r *repository) BeginTxx(ctx context.Context) (*sqlx.Tx, error) {
	return r.db.BeginTxx(ctx, nil)
}

func (r *repository) NextRobokassaInvID(ctx context.Context) (int64, error) {
	id, err := r.nextRobokassaInvID(ctx)
	if err == nil {
		return id, nil
	}
	if !isMissingRobokassaSequenceErr(err) {
		return 0, err
	}
	if createErr := r.createRobokassaSequenceIfMissing(ctx); createErr != nil {
		fallbackID, fallbackErr := r.nextRobokassaInvIDFromPayments(ctx)
		if fallbackErr != nil {
			return 0, fmt.Errorf("%w; fallback invoice id generation failed: %v", createErr, fallbackErr)
		}
		return fallbackID, nil
	}
	id, err = r.nextRobokassaInvID(ctx)
	if err == nil {
		return id, nil
	}
	fallbackID, fallbackErr := r.nextRobokassaInvIDFromPayments(ctx)
	if fallbackErr != nil {
		return generateFallbackRobokassaInvID(), nil
	}
	return fallbackID, nil
}

func (r *repository) nextRobokassaInvID(ctx context.Context) (int64, error) {
	var id int64
	if err := r.db.QueryRowContext(ctx, `SELECT nextval('robokassa_invoice_seq')`).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}
func (r *repository) createRobokassaSequenceIfMissing(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `CREATE SEQUENCE IF NOT EXISTS robokassa_invoice_seq START WITH 1000 INCREMENT BY 1 NO MAXVALUE NO MINVALUE CACHE 1`)
	if err != nil {
		return fmt.Errorf("failed to auto-create robokassa sequence: %w", err)
	}
	return nil
}

func (r *repository) nextRobokassaInvIDFromPayments(ctx context.Context) (int64, error) {
	var id int64
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(robokassa_inv_id), 999) + 1 FROM payments`).Scan(&id); err != nil {
		if isUndefinedColumnErr(err, "robokassa_inv_id") {
			if legacyID, legacyErr := r.nextRobokassaInvIDFromLegacyInvID(ctx); legacyErr == nil {
				return legacyID, nil
			}
			return generateFallbackRobokassaInvID(), nil
		}
		return 0, err
	}
	if id <= 0 {
		return 1000, nil
	}
	return id, nil
}

func (r *repository) nextRobokassaInvIDFromLegacyInvID(ctx context.Context) (int64, error) {
	var id int64
	query := `SELECT COALESCE(MAX(CASE WHEN inv_id ~ '^[0-9]+$' THEN inv_id::bigint ELSE NULL END), 999) + 1 FROM payments`
	if err := r.db.QueryRowContext(ctx, query).Scan(&id); err != nil {
		if isUndefinedColumnErr(err, "inv_id") {
			return 0, err
		}
		return 0, err
	}
	if id <= 0 {
		return 1000, nil
	}
	return id, nil
}

func generateFallbackRobokassaInvID() int64 {
	// Millisecond timestamp keeps the value positive and practically unique for degraded mode.
	return time.Now().UnixMilli()
}

func isMissingRobokassaSequenceErr(err error) bool {
	if err == nil {
		return false
	}
	errText := strings.ToLower(err.Error())
	return strings.Contains(errText, "robokassa_invoice_seq") && strings.Contains(errText, "does not exist")
}

func isUndefinedPaymentsColumnErr(err error) bool {
	if err == nil {
		return false
	}
	errText := strings.ToLower(err.Error())
	if !(strings.Contains(errText, "column") && strings.Contains(errText, "does not exist")) {
		return false
	}
	return strings.Contains(errText, "payments")
}

func isUndefinedColumnErr(err error, column string) bool {
	if err == nil {
		return false
	}
	errText := strings.ToLower(err.Error())
	if !strings.Contains(errText, "column") || !strings.Contains(errText, "does not exist") {
		return false
	}
	column = strings.ToLower(strings.TrimSpace(column))
	if column == "" {
		return false
	}

	columnPatterns := []string{
		"." + column,
		"\"" + column + "\"",
		" " + column + " ",
		" " + column + " does not exist",
	}
	for _, pattern := range columnPatterns {
		if strings.Contains(errText, pattern) {
			return true
		}
	}
	return false
}
