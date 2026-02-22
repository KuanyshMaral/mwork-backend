package subscription

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines subscription data access
type Repository interface {
	// Plans
	GetPlanByID(ctx context.Context, id PlanID) (*Plan, error)
	ListPlans(ctx context.Context) ([]*Plan, error)

	// Subscriptions
	Create(ctx context.Context, sub *Subscription) error
	GetByID(ctx context.Context, id uuid.UUID) (*Subscription, error)
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*Subscription, error)
	Update(ctx context.Context, sub *Subscription) error
	Cancel(ctx context.Context, id uuid.UUID, reason string) error
	ExpireOldSubscriptions(ctx context.Context) (int, error)

	GetUserRole(ctx context.Context, userID uuid.UUID) (Audience, error)
	GetLimitOverrideTotal(ctx context.Context, userID uuid.UUID, limitKey string) (int, error)
	CreateLimitOverride(ctx context.Context, override *LimitOverride) error
	GetAllLimitOverrides(ctx context.Context, userID uuid.UUID) (map[string]int, error)
}

type repository struct {
	db *sqlx.DB
}

// NewRepository creates subscription repository
func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

// Plans

func (r *repository) GetPlanByID(ctx context.Context, id PlanID) (*Plan, error) {
	query := `
		SELECT
			id, name, description, price_monthly, price_yearly,
			audience, is_active, created_at,
			monthly_consumables, features_and_quotas
		FROM plans
		WHERE id = $1 AND is_active = true
	`
	var plan Plan
	err := r.db.GetContext(ctx, &plan, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	plan.ParseJSONB()
	return &plan, nil
}

func (r *repository) ListPlans(ctx context.Context) ([]*Plan, error) {
	query := `
		SELECT
			id, name, description, price_monthly, price_yearly,
			audience, is_active, created_at,
			monthly_consumables, features_and_quotas
		FROM plans
		WHERE is_active = true
		ORDER BY price_monthly
	`
	var plans []*Plan
	if err := r.db.SelectContext(ctx, &plans, query); err != nil {
		return nil, err
	}
	for _, p := range plans {
		p.ParseJSONB()
	}
	return plans, nil
}

// Subscriptions

func (r *repository) Create(ctx context.Context, sub *Subscription) error {
	query := `
		INSERT INTO subscriptions (id, user_id, plan_id, started_at, expires_at, status, billing_period, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		sub.ID,
		sub.UserID,
		sub.PlanID,
		sub.StartedAt,
		sub.ExpiresAt,
		sub.Status,
		sub.BillingPeriod,
		sub.CreatedAt,
		sub.UpdatedAt,
	)
	return err
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Subscription, error) {
	query := `
		SELECT
			id, user_id, plan_id, started_at, expires_at, status,
			cancelled_at, cancel_reason, billing_period, created_at, updated_at
		FROM subscriptions
		WHERE id = $1
	`
	var sub Subscription
	err := r.db.GetContext(ctx, &sub, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func (r *repository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*Subscription, error) {
	query := `
		SELECT
			id, user_id, plan_id, started_at, expires_at, status,
			cancelled_at, cancel_reason, billing_period, created_at, updated_at
		FROM subscriptions
		WHERE user_id = $1 AND status = 'active'
		ORDER BY created_at DESC
		LIMIT 1
	`
	var sub Subscription
	err := r.db.GetContext(ctx, &sub, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func (r *repository) Update(ctx context.Context, sub *Subscription) error {
	query := `
		UPDATE subscriptions SET
			plan_id = $2, expires_at = $3, status = $4, billing_period = $5, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		sub.ID,
		sub.PlanID,
		sub.ExpiresAt,
		sub.Status,
		sub.BillingPeriod,
	)
	return err
}

func (r *repository) Cancel(ctx context.Context, id uuid.UUID, reason string) error {
	query := `
		UPDATE subscriptions SET
			status = 'cancelled', cancelled_at = NOW(), cancel_reason = $2, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, reason)
	return err
}

func (r *repository) ExpireOldSubscriptions(ctx context.Context) (int, error) {
	query := `
		UPDATE subscriptions SET
			status = 'expired', updated_at = NOW()
		WHERE status = 'active' AND expires_at IS NOT NULL AND expires_at < NOW()
	`
	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	affected, _ := result.RowsAffected()
	return int(affected), nil
}

func (r *repository) GetUserRole(ctx context.Context, userID uuid.UUID) (Audience, error) {
	var role string
	if err := r.db.GetContext(ctx, &role, `SELECT role FROM users WHERE id = $1`, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrSubscriptionNotFound
		}
		return "", err
	}
	switch role {
	case "model":
		return AudienceModel, nil
	case "employer", "agency":
		return AudienceEmployer, nil
	default:
		return "", fmt.Errorf("unsupported role for subscriptions: %s", role)
	}
}

func (r *repository) GetLimitOverrideTotal(ctx context.Context, userID uuid.UUID, limitKey string) (int, error) {
	var total int
	err := r.db.GetContext(ctx, &total, `
		SELECT COALESCE(SUM(delta), 0)
		FROM limit_overrides
		WHERE user_id = $1 AND limit_key = $2
	`, userID, limitKey)
	return total, err
}

func (r *repository) CreateLimitOverride(ctx context.Context, override *LimitOverride) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO limit_overrides (id, user_id, limit_key, delta, reason, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, override.ID, override.UserID, override.LimitKey, override.Delta, override.Reason, override.CreatedBy, override.CreatedAt)
	return err
}

func (r *repository) GetAllLimitOverrides(ctx context.Context, userID uuid.UUID) (map[string]int, error) {
	type row struct {
		LimitKey string `db:"limit_key"`
		Total    int    `db:"total"`
	}
	items := []row{}
	err := r.db.SelectContext(ctx, &items, `
		SELECT limit_key, COALESCE(SUM(delta), 0) AS total
		FROM limit_overrides
		WHERE user_id = $1
		GROUP BY limit_key
	`, userID)
	if err != nil {
		return nil, err
	}
	result := make(map[string]int, len(items))
	for _, item := range items {
		result[item.LimitKey] = item.Total
	}
	return result, nil
}
