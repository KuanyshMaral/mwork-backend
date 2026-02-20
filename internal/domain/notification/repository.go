package notification

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines notification data access
type Repository interface {
	Create(ctx context.Context, n *Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*Notification, error)
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int, unreadOnly bool) ([]*Notification, error)
	CountUnreadByUser(ctx context.Context, userID uuid.UUID) (int, error)
	MarkAsRead(ctx context.Context, userID, id uuid.UUID) (bool, error)
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int64, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteOldByUser(ctx context.Context, userID uuid.UUID, days int) (int, error)
	DeleteOlderThan(ctx context.Context, age time.Duration) (int64, error)
}

type repository struct {
	db *sqlx.DB
}

// NewRepository creates notification repository
func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, n *Notification) error {
	query := `
		INSERT INTO notifications (id, user_id, type, title, body, data, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		n.ID,
		n.UserID,
		n.Type,
		n.Title,
		n.Body,
		n.Data,
		n.IsRead,
		n.CreatedAt,
	)
	return err
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Notification, error) {
	query := `SELECT * FROM notifications WHERE id = $1`
	var n Notification
	err := r.db.GetContext(ctx, &n, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &n, nil
}

func buildListByUserQuery(unreadOnly bool) string {
	query := `
		SELECT * FROM notifications 
		WHERE user_id = $1
	`
	if unreadOnly {
		query += ` AND is_read = false`
	}
	query += `
		ORDER BY created_at DESC 
		LIMIT $2 OFFSET $3
	`
	return query
}

func (r *repository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int, unreadOnly bool) ([]*Notification, error) {
	query := buildListByUserQuery(unreadOnly)
	var notifications []*Notification
	err := r.db.SelectContext(ctx, &notifications, query, userID, limit, offset)
	return notifications, err
}

func (r *repository) CountUnreadByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND NOT is_read`
	var count int
	err := r.db.GetContext(ctx, &count, query, userID)
	return count, err
}

func (r *repository) MarkAsRead(ctx context.Context, userID, id uuid.UUID) (bool, error) {
	query := `UPDATE notifications SET is_read = true, read_at = NOW() WHERE id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *repository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	query := `UPDATE notifications SET is_read = true, read_at = NOW() WHERE user_id = $1 AND NOT is_read`
	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Delete removes a notification
func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM notifications WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// DeleteOldByUser removes old notifications for a user (older than X days)
func (r *repository) DeleteOldByUser(ctx context.Context, userID uuid.UUID, days int) (int, error) {
	query := `DELETE FROM notifications WHERE user_id = $1 AND created_at < NOW() - ($2 || ' days')::INTERVAL`
	result, err := r.db.ExecContext(ctx, query, userID, days)
	if err != nil {
		return 0, err
	}
	affected, _ := result.RowsAffected()
	return int(affected), nil
}

// DeleteOlderThan removes all notifications older than the specified duration
func (r *repository) DeleteOlderThan(ctx context.Context, age time.Duration) (int64, error) {
	cutoff := time.Now().Add(-age)
	query := `DELETE FROM notifications WHERE created_at < $1`
	result, err := r.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
