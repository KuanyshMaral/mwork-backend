package user

import (
	"context"
	"database/sql"
	"errors"
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
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID, ip string) error
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
		INSERT INTO users (id, email, password_hash, role, email_verified, is_banned, credit_balance)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
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

	return err
}

// GetByID returns user by ID
func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, email, password_hash, role, email_verified, is_banned, credit_balance,
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
		SELECT id, email, password_hash, role, email_verified, is_banned, credit_balance,
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

	return err
}

// UpdateEmailVerified updates email verified status
func (r *repository) UpdateEmailVerified(ctx context.Context, id uuid.UUID, verified bool) error {
	query := `UPDATE users SET email_verified = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, verified)
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
