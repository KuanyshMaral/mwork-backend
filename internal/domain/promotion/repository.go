package promotion

import (
	"context"
	"database/sql"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Repository handles promotion database operations
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new promotion repository
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// GetProfileIDByUserID returns profile ID for authenticated user
func (r *Repository) GetProfileIDByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var profileID uuid.UUID
	err := r.db.GetContext(ctx, &profileID, `
		SELECT id
		FROM profiles
		WHERE user_id = $1
	`, userID)
	if err == sql.ErrNoRows {
		return uuid.Nil, ErrProfileNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}

	return profileID, nil
}

// Create inserts a new promotion
func (r *Repository) Create(ctx context.Context, p *Promotion) error {
	query := `
		INSERT INTO profile_promotions (
			id, profile_id, title, description, photo_url, specialization,
			target_audience, target_cities, budget_amount, daily_budget,
			duration_days, status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		p.ID,
		p.ProfileID,
		p.Title,
		p.Description,
		p.PhotoURL,
		p.Specialization,
		p.TargetAudience,
		pq.Array(p.TargetCities),
		p.BudgetAmount,
		p.DailyBudget,
		p.DurationDays,
		p.Status,
		p.CreatedAt,
		p.UpdatedAt,
	)
	return err
}

// GetByID returns a promotion by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Promotion, error) {
	query := `
		SELECT id, profile_id, title, description, photo_url, specialization,
			target_audience, target_cities, budget_amount, daily_budget,
			duration_days, status, starts_at, ends_at, impressions, clicks,
			responses, spent_amount, payment_id, created_at, updated_at
		FROM profile_promotions
		WHERE id = $1
	`

	var p Promotion
	var targetCities pq.StringArray

	row := r.db.QueryRowContext(ctx, query, id)
	var paymentID sql.NullString
	err := row.Scan(
		&p.ID, &p.ProfileID, &p.Title, &p.Description, &p.PhotoURL, &p.Specialization,
		&p.TargetAudience, &targetCities, &p.BudgetAmount, &p.DailyBudget,
		&p.DurationDays, &p.Status, &p.StartsAt, &p.EndsAt, &p.Impressions, &p.Clicks,
		&p.Responses, &p.SpentAmount, &paymentID, &p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrPromotionNotFound
	}
	if err != nil {
		return nil, err
	}

	p.TargetCities = targetCities
	if paymentID.Valid {
		parsedPaymentID, parseErr := uuid.Parse(strings.TrimSpace(paymentID.String))
		if parseErr != nil {
			return nil, parseErr
		}
		p.PaymentID = &parsedPaymentID
	}
	return &p, nil
}

// GetByProfileID returns all promotions for a profile
func (r *Repository) GetByProfileID(ctx context.Context, profileID uuid.UUID) ([]Promotion, error) {
	query := `
		SELECT id, profile_id, title, description, photo_url, specialization,
			target_audience, target_cities, budget_amount, daily_budget,
			duration_days, status, starts_at, ends_at, impressions, clicks,
			responses, spent_amount, payment_id, created_at, updated_at
		FROM profile_promotions
		WHERE profile_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var promotions []Promotion
	for rows.Next() {
		var p Promotion
		var targetCities pq.StringArray
		var paymentID sql.NullString

		err := rows.Scan(
			&p.ID, &p.ProfileID, &p.Title, &p.Description, &p.PhotoURL, &p.Specialization,
			&p.TargetAudience, &targetCities, &p.BudgetAmount, &p.DailyBudget,
			&p.DurationDays, &p.Status, &p.StartsAt, &p.EndsAt, &p.Impressions, &p.Clicks,
			&p.Responses, &p.SpentAmount, &paymentID, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		p.TargetCities = targetCities
		if paymentID.Valid {
			parsedPaymentID, parseErr := uuid.Parse(strings.TrimSpace(paymentID.String))
			if parseErr != nil {
				return nil, parseErr
			}
			p.PaymentID = &parsedPaymentID
		}
		promotions = append(promotions, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return promotions, nil
}

// UpdateStatus updates promotion status
func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error {
	query := `UPDATE profile_promotions SET status = $2, updated_at = NOW() WHERE id = $1`
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

// Activate activates a promotion with start/end times
func (r *Repository) Activate(ctx context.Context, id uuid.UUID, startsAt, endsAt sql.NullTime, paymentID *uuid.UUID) error {
	query := `
		UPDATE profile_promotions 
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

// IncrementStats increments promotion analytics
func (r *Repository) IncrementStats(ctx context.Context, id uuid.UUID, impressions, clicks, responses int, spent int64) error {
	query := `
		UPDATE profile_promotions 
		SET impressions = impressions + $2, 
			clicks = clicks + $3, 
			responses = responses + $4, 
			spent_amount = spent_amount + $5,
			updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, impressions, clicks, responses, spent)
	return err
}

// GetDailyStats returns daily stats for a promotion
func (r *Repository) GetDailyStats(ctx context.Context, promotionID uuid.UUID) ([]DailyStats, error) {
	query := `
		SELECT id, promotion_id, date, impressions, clicks, responses, spent
		FROM promotion_daily_stats
		WHERE promotion_id = $1
		ORDER BY date DESC
	`

	var stats []DailyStats
	err := r.db.SelectContext(ctx, &stats, query, promotionID)
	return stats, err
}

// GetActivePromotions returns all currently active promotions
func (r *Repository) GetActivePromotions(ctx context.Context) ([]Promotion, error) {
	query := `
		SELECT id, profile_id, title, description, photo_url, specialization,
			target_audience, target_cities, budget_amount, daily_budget,
			duration_days, status, starts_at, ends_at, impressions, clicks,
			responses, spent_amount, payment_id, created_at, updated_at
		FROM profile_promotions
		WHERE status = 'active' AND NOW() BETWEEN starts_at AND ends_at
		ORDER BY budget_amount DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var promotions []Promotion
	for rows.Next() {
		var p Promotion
		var targetCities pq.StringArray
		var paymentID sql.NullString

		err := rows.Scan(
			&p.ID, &p.ProfileID, &p.Title, &p.Description, &p.PhotoURL, &p.Specialization,
			&p.TargetAudience, &targetCities, &p.BudgetAmount, &p.DailyBudget,
			&p.DurationDays, &p.Status, &p.StartsAt, &p.EndsAt, &p.Impressions, &p.Clicks,
			&p.Responses, &p.SpentAmount, &paymentID, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		p.TargetCities = targetCities
		if paymentID.Valid {
			parsedPaymentID, parseErr := uuid.Parse(strings.TrimSpace(paymentID.String))
			if parseErr != nil {
				return nil, parseErr
			}
			p.PaymentID = &parsedPaymentID
		}
		promotions = append(promotions, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return promotions, nil
}
