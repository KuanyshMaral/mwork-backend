package auth

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type RefreshTokenRecord struct {
	ID        uuid.UUID    `db:"id"`
	UserID    uuid.UUID    `db:"user_id"`
	TokenHash string       `db:"token_hash"`
	JTI       string       `db:"jti"`
	ExpiresAt time.Time    `db:"expires_at"`
	UsedAt    sql.NullTime `db:"used_at"`
	RevokedAt sql.NullTime `db:"revoked_at"`
	CreatedAt time.Time    `db:"created_at"`
	UserAgent string       `db:"user_agent"`
	IP        string       `db:"ip"`
}

type RefreshTokenRepository struct {
	db *sqlx.DB
}

func NewRefreshTokenRepository(db *sqlx.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, rec *RefreshTokenRecord) error {
	query := `
		INSERT INTO user_refresh_tokens (id, user_id, token_hash, jti, expires_at, used_at, revoked_at, created_at, user_agent, ip)
		VALUES ($1, $2, $3, $4, $5, NULL, NULL, NOW(), $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query, rec.ID, rec.UserID, rec.TokenHash, rec.JTI, rec.ExpiresAt, rec.UserAgent, rec.IP)
	return err
}

func (r *RefreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*RefreshTokenRecord, error) {
	query := `
		SELECT id, user_id, token_hash, jti, expires_at, used_at, revoked_at, created_at, user_agent, ip
		FROM user_refresh_tokens
		WHERE token_hash = $1
	`
	var rec RefreshTokenRecord
	if err := r.db.GetContext(ctx, &rec, query, tokenHash); err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *RefreshTokenRepository) MarkUsed(ctx context.Context, tokenHash string) error {
	query := `UPDATE user_refresh_tokens SET used_at = NOW() WHERE token_hash = $1 AND used_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, tokenHash)
	return err
}

func (r *RefreshTokenRepository) RevokeByTokenHash(ctx context.Context, tokenHash string) error {
	query := `UPDATE user_refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, tokenHash)
	return err
}

func (r *RefreshTokenRepository) RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE user_refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}
