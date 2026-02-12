package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const (
	verificationCodeTTL         = 5 * time.Minute
	verificationCodeMaxAttempts = 5
)

type VerificationCodeRecord struct {
	UserID    uuid.UUID  `db:"user_id"`
	CodeHash  string     `db:"code_hash"`
	Attempts  int        `db:"attempts"`
	ExpiresAt time.Time  `db:"expires_at"`
	UsedAt    *time.Time `db:"used_at"`
	CreatedAt time.Time  `db:"created_at"`
}

type VerificationCodeRepository struct {
	db *sqlx.DB
}

func NewVerificationCodeRepository(db *sqlx.DB) *VerificationCodeRepository {
	return &VerificationCodeRepository{db: db}
}

func (r *VerificationCodeRepository) Upsert(ctx context.Context, userID uuid.UUID, codeHash string, expiresAt time.Time) error {
	query := `
		INSERT INTO user_verification_codes (user_id, code_hash, attempts, expires_at, used_at, created_at)
		VALUES ($1, $2, 0, $3, NULL, NOW())
		ON CONFLICT (user_id)
		DO UPDATE SET code_hash = EXCLUDED.code_hash,
		              attempts = 0,
		              expires_at = EXCLUDED.expires_at,
		              used_at = NULL,
		              created_at = NOW()
	`
	_, err := r.db.ExecContext(ctx, query, userID, codeHash, expiresAt)
	return err
}

func (r *VerificationCodeRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*VerificationCodeRecord, error) {
	query := `
		SELECT user_id, code_hash, attempts, expires_at, used_at, created_at
		FROM user_verification_codes
		WHERE user_id = $1
	`
	var rec VerificationCodeRecord
	if err := r.db.GetContext(ctx, &rec, query, userID); err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *VerificationCodeRepository) IncrementAttempts(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `
		UPDATE user_verification_codes
		SET attempts = attempts + 1
		WHERE user_id = $1
		RETURNING attempts
	`
	var attempts int
	if err := r.db.QueryRowContext(ctx, query, userID).Scan(&attempts); err != nil {
		return 0, err
	}
	return attempts, nil
}

func (r *VerificationCodeRepository) Invalidate(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE user_verification_codes
		SET used_at = NOW()
		WHERE user_id = $1
	`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

func (r *VerificationCodeRepository) MarkUsed(ctx context.Context, userID uuid.UUID) error {
	return r.Invalidate(ctx, userID)
}
