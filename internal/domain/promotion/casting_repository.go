package promotion

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// CastingRepository handles casting promotion DB operations
type CastingRepository struct {
	db *sqlx.DB
}

// NewCastingRepository creates a new casting promotion repository
func NewCastingRepository(db *sqlx.DB) *CastingRepository {
	return &CastingRepository{db: db}
}

// GetEmployerIDByUserID returns the employer profile ID for a user
func (r *CastingRepository) GetEmployerIDByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var profileID uuid.UUID
	err := r.db.GetContext(ctx, &profileID, `
		SELECT id FROM employer_profiles WHERE user_id = $1
	`, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, ErrProfileNotFound
		}
		return uuid.Nil, err
	}
	return profileID, nil
}

// VerifyCastingOwner checks if the employer owns the given casting
func (r *CastingRepository) VerifyCastingOwner(ctx context.Context, castingID, employerID uuid.UUID) error {
	var count int
	err := r.db.GetContext(ctx, &count, `
		SELECT COUNT(*) FROM castings WHERE id = $1 AND creator_id = $2
	`, castingID, employerID)
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrNotPromotionOwner
	}
	return nil
}

// Create inserts a new casting promotion
func (r *CastingRepository) Create(ctx context.Context, cp *CastingPromotion) error {
	query := `
		INSERT INTO casting_promotions (
			id, casting_id, employer_id, custom_title, custom_photo_url,
			budget_amount, daily_budget, duration_days, status,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`
	_, err := r.db.ExecContext(ctx, query,
		cp.ID, cp.CastingID, cp.EmployerID, cp.CustomTitle, cp.CustomPhotoURL,
		cp.BudgetAmount, cp.DailyBudget, cp.DurationDays, cp.Status,
		cp.CreatedAt, cp.UpdatedAt,
	)
	return err
}

// GetByID returns a casting promotion by its ID
func (r *CastingRepository) GetByID(ctx context.Context, id uuid.UUID) (*CastingPromotion, error) {
	query := `
		SELECT id, casting_id, employer_id, custom_title, custom_photo_url,
			budget_amount, daily_budget, duration_days, status,
			starts_at, ends_at, impressions, clicks, responses,
			spent_amount, payment_id, created_at, updated_at
		FROM casting_promotions WHERE id = $1
	`
	var cp CastingPromotion
	var paymentID uuid.UUID
	var paymentIDNull sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&cp.ID, &cp.CastingID, &cp.EmployerID, &cp.CustomTitle, &cp.CustomPhotoURL,
		&cp.BudgetAmount, &cp.DailyBudget, &cp.DurationDays, &cp.Status,
		&cp.StartsAt, &cp.EndsAt, &cp.Impressions, &cp.Clicks, &cp.Responses,
		&cp.SpentAmount, &paymentIDNull, &cp.CreatedAt, &cp.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPromotionNotFound
		}
		return nil, err
	}
	if paymentIDNull.Valid {
		if pid, err := uuid.Parse(paymentIDNull.String); err == nil {
			paymentID = pid
			cp.PaymentID = &paymentID
		}
	}
	return &cp, nil
}

// GetByCastingID returns all promotions for a given casting
func (r *CastingRepository) GetByCastingID(ctx context.Context, castingID uuid.UUID) ([]CastingPromotion, error) {
	query := `
		SELECT id, casting_id, employer_id, custom_title, custom_photo_url,
			budget_amount, daily_budget, duration_days, status,
			starts_at, ends_at, impressions, clicks, responses,
			spent_amount, payment_id, created_at, updated_at
		FROM casting_promotions WHERE casting_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, castingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var promotions []CastingPromotion
	for rows.Next() {
		var cp CastingPromotion
		var paymentIDNull sql.NullString
		if err := rows.Scan(
			&cp.ID, &cp.CastingID, &cp.EmployerID, &cp.CustomTitle, &cp.CustomPhotoURL,
			&cp.BudgetAmount, &cp.DailyBudget, &cp.DurationDays, &cp.Status,
			&cp.StartsAt, &cp.EndsAt, &cp.Impressions, &cp.Clicks, &cp.Responses,
			&cp.SpentAmount, &paymentIDNull, &cp.CreatedAt, &cp.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if paymentIDNull.Valid {
			if pid, err := uuid.Parse(paymentIDNull.String); err == nil {
				cp.PaymentID = &pid
			}
		}
		promotions = append(promotions, cp)
	}
	return promotions, rows.Err()
}

// GetByEmployerID returns all promotions created by an employer
func (r *CastingRepository) GetByEmployerID(ctx context.Context, employerID uuid.UUID) ([]CastingPromotion, error) {
	query := `
		SELECT id, casting_id, employer_id, custom_title, custom_photo_url,
			budget_amount, daily_budget, duration_days, status,
			starts_at, ends_at, impressions, clicks, responses,
			spent_amount, payment_id, created_at, updated_at
		FROM casting_promotions WHERE employer_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, employerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var promotions []CastingPromotion
	for rows.Next() {
		var cp CastingPromotion
		var paymentIDNull sql.NullString
		if err := rows.Scan(
			&cp.ID, &cp.CastingID, &cp.EmployerID, &cp.CustomTitle, &cp.CustomPhotoURL,
			&cp.BudgetAmount, &cp.DailyBudget, &cp.DurationDays, &cp.Status,
			&cp.StartsAt, &cp.EndsAt, &cp.Impressions, &cp.Clicks, &cp.Responses,
			&cp.SpentAmount, &paymentIDNull, &cp.CreatedAt, &cp.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if paymentIDNull.Valid {
			if pid, err := uuid.Parse(paymentIDNull.String); err == nil {
				cp.PaymentID = &pid
			}
		}
		promotions = append(promotions, cp)
	}
	return promotions, rows.Err()
}

// Activate activates a casting promotion and sets its schedule
func (r *CastingRepository) Activate(ctx context.Context, id uuid.UUID, startsAt, endsAt sql.NullTime, paymentID *uuid.UUID) error {
	query := `
		UPDATE casting_promotions
		SET status = 'active', starts_at = $2, ends_at = $3, payment_id = $4, updated_at = NOW()
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query, id, startsAt, endsAt, paymentID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrPromotionNotFound
	}
	return nil
}

// UpdateStatus updates the status of a casting promotion
func (r *CastingRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error {
	query := `UPDATE casting_promotions SET status = $2, updated_at = NOW() WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id, status)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrPromotionNotFound
	}
	return nil
}

// GetDailyStats returns daily stats for a casting promotion
func (r *CastingRepository) GetDailyStats(ctx context.Context, promotionID uuid.UUID) ([]CastingPromotionDailyStats, error) {
	query := `
		SELECT id, promotion_id, date, impressions, clicks, responses, spent
		FROM casting_promotion_stats
		WHERE promotion_id = $1
		ORDER BY date DESC
	`
	var stats []CastingPromotionDailyStats
	err := r.db.SelectContext(ctx, &stats, query, promotionID)
	return stats, err
}

// ExpireCompleted moves all promotions past their end date to 'completed'
func (r *CastingRepository) ExpireCompleted(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE casting_promotions 
		SET status = 'completed', updated_at = NOW()
		WHERE status = 'active' AND ends_at < $1
	`, time.Now())
	if err != nil {
		return 0, err
	}
	count, _ := result.RowsAffected()
	return count, nil
}
