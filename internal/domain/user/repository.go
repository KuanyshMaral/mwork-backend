package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines user data access interface
type Repository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateEmailVerified(ctx context.Context, id uuid.UUID, verified bool) error
	UpdateVerificationFlags(ctx context.Context, id uuid.UUID, emailVerified bool, isVerified bool) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID, ip string) error

	// Connects (Two-Buckets) — Model response connects
	// DeductModelConnect atomically deducts 1 from free_response_connects first,
	// then purchased_response_connects if free is 0. Returns ErrInsufficientConnects if both are 0.
	DeductModelConnect(ctx context.Context, userID uuid.UUID) error
	// RefreshModelConnectsIfNeeded resets free connects if the last reset was before this month.
	RefreshModelConnectsIfNeeded(ctx context.Context, userID uuid.UUID, freeLimit int) error
	// GetConnectsBalance returns current free+purchased connect amounts.
	GetConnectsBalance(ctx context.Context, userID uuid.UUID) (free int, purchased int, err error)
	// AddPurchasedModelConnects adds to the purchased (permanent) connects bucket.
	AddPurchasedModelConnects(ctx context.Context, userID uuid.UUID, amount int) error
}

// repository implements Repository
type repository struct {
	db *sqlx.DB
}

// NewRepository creates new user repository
func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

// Create creates a new user
func (r *repository) Create(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (id, email, password_hash, role, email_verified, is_verified, is_banned, credit_balance)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.EmailVerified,
		user.IsVerified,
		user.IsBanned,
		user.CreditBalance,
	)
	if err != nil {
		return fmt.Errorf("user repository create: %w", err)
	}

	return nil
}

// GetByID returns user by ID
func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, email, password_hash, role, email_verified, is_verified, is_banned, credit_balance,
		       user_verification_status,
		       created_at, updated_at
		FROM users WHERE id = $1
	`
	var user User
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

// GetByEmail returns user by email
func (r *repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, password_hash, role, email_verified, is_verified, is_banned, credit_balance,
		       user_verification_status,
		       created_at, updated_at
		FROM users WHERE email = $1
	`
	var user User
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

// Update updates user
func (r *repository) Update(ctx context.Context, user *User) error {
	query := `
		UPDATE users 
		SET email = $2, password_hash = $3, 
		    role = $4, email_verified = $5, is_banned = $6, credit_balance = $7, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.EmailVerified,
		user.IsBanned,
		user.CreditBalance,
	)
	if err != nil {
		return fmt.Errorf("user repository update: %w", err)
	}

	return nil
}

// UpdateEmailVerified updates email verified status
func (r *repository) UpdateEmailVerified(ctx context.Context, id uuid.UUID, verified bool) error {
	query := `UPDATE users SET email_verified = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, verified)
	return err
}

func (r *repository) UpdateVerificationFlags(ctx context.Context, id uuid.UUID, emailVerified bool, isVerified bool) error {
	query := `UPDATE users SET email_verified = $2, is_verified = $3, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, emailVerified, isVerified)
	return err
}

// UpdatePassword updates user password hash
func (r *repository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, passwordHash)
	return err
}

// UpdateStatus updates user status (bans/unbans)
func (r *repository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error {
	// Map status to is_banned
	isBanned := status == StatusBanned || status == StatusSuspended
	query := `UPDATE users SET is_banned = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, isBanned)
	return err
}

// UpdateLastLogin updates last login timestamp and IP
func (r *repository) UpdateLastLogin(ctx context.Context, id uuid.UUID, ip string) error {
	query := `UPDATE users SET last_login_at = $2, last_login_ip = $3, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, time.Now(), ip)
	return err
}

// Delete soft deletes a user by banning
func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET is_banned = true, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// ErrInsufficientConnects is returned when both free and purchased balances are 0.
var ErrInsufficientConnects = errors.New("insufficient connects")

// DeductModelConnect atomically deducts 1 connect using the Two-Buckets pattern:
// 1. Try to deduct from model_free_response_connects.
// 2. If 0, try to deduct from model_purchased_response_connects.
// 3. If both 0, return ErrInsufficientConnects.
func (r *repository) DeductModelConnect(ctx context.Context, userID uuid.UUID) error {
	// Try free bucket first
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET model_free_response_connects = model_free_response_connects - 1
		 WHERE id = $1 AND model_free_response_connects > 0`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("deduct free connect: %w", err)
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil // deducted from free bucket
	}

	// Free is empty — try purchased bucket
	res, err = r.db.ExecContext(ctx,
		`UPDATE users SET model_purchased_response_connects = model_purchased_response_connects - 1
		 WHERE id = $1 AND model_purchased_response_connects > 0`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("deduct purchased connect: %w", err)
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil // deducted from purchased bucket
	}

	return ErrInsufficientConnects
}

// RefreshModelConnectsIfNeeded lazily refreshes free connects at the start of each calendar month.
func (r *repository) RefreshModelConnectsIfNeeded(ctx context.Context, userID uuid.UUID, freeLimit int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users
		 SET model_free_response_connects = $2,
		     model_connects_reset_at = NOW()
		 WHERE id = $1
		   AND (model_connects_reset_at IS NULL
		        OR model_connects_reset_at < DATE_TRUNC('month', NOW()))`,
		userID, freeLimit,
	)
	return err
}

// GetConnectsBalance returns current free and purchased connect balances.
func (r *repository) GetConnectsBalance(ctx context.Context, userID uuid.UUID) (int, int, error) {
	var free, purchased int
	err := r.db.QueryRowContext(ctx,
		`SELECT model_free_response_connects, model_purchased_response_connects FROM users WHERE id = $1`,
		userID,
	).Scan(&free, &purchased)
	if err != nil {
		return 0, 0, err
	}
	return free, purchased, nil
}

// AddPurchasedModelConnects adds connects to the permanent purchased bucket.
func (r *repository) AddPurchasedModelConnects(ctx context.Context, userID uuid.UUID, amount int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET model_purchased_response_connects = model_purchased_response_connects + $2 WHERE id = $1`,
		userID, amount,
	)
	return err
}
