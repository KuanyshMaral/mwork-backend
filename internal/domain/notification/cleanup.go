package notification

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

// CleanupJob handles notification retention cleanup
type CleanupJob struct {
	db            *sqlx.DB
	retentionDays int
}

// NewCleanupJob creates a cleanup job
func NewCleanupJob(db *sqlx.DB, retentionDays int) *CleanupJob {
	if retentionDays <= 0 {
		retentionDays = 90 // Default 90 days
	}
	return &CleanupJob{
		db:            db,
		retentionDays: retentionDays,
	}
}

// Start starts the cleanup job with the given interval
func (j *CleanupJob) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run immediately on start
	j.run(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Notification cleanup job stopped")
			return
		case <-ticker.C:
			j.run(ctx)
		}
	}
}

// run executes the cleanup
func (j *CleanupJob) run(ctx context.Context) {
	cutoff := time.Now().AddDate(0, 0, -j.retentionDays)

	result, err := j.db.ExecContext(ctx, `
		DELETE FROM notifications 
		WHERE created_at < $1 AND is_read = true
	`, cutoff)

	if err != nil {
		log.Error().Err(err).Msg("Failed to cleanup old notifications")
		return
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		log.Info().
			Int64("deleted", rows).
			Int("retention_days", j.retentionDays).
			Msg("Cleaned up old notifications")
	}

	// Also cleanup old unread (older than 180 days)
	unreadCutoff := time.Now().AddDate(0, 0, -180)
	result2, _ := j.db.ExecContext(ctx, `
		DELETE FROM notifications 
		WHERE created_at < $1
	`, unreadCutoff)

	if unreadRows, _ := result2.RowsAffected(); unreadRows > 0 {
		log.Info().Int64("deleted_unread", unreadRows).Msg("Cleaned up very old unread notifications")
	}

	// Cleanup old notification groups
	j.db.ExecContext(ctx, `
		DELETE FROM notification_groups 
		WHERE created_at < $1
	`, cutoff)

	// Cleanup inactive device tokens (not used for 90 days)
	j.db.ExecContext(ctx, `
		DELETE FROM device_tokens 
		WHERE last_used_at < $1 OR is_active = false
	`, cutoff)
}

// RunOnce runs cleanup once (for manual trigger or testing)
func (j *CleanupJob) RunOnce(ctx context.Context) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -j.retentionDays)

	result, err := j.db.ExecContext(ctx, `
		DELETE FROM notifications 
		WHERE created_at < $1 AND is_read = true
	`, cutoff)

	if err != nil {
		return 0, err
	}

	rows, _ := result.RowsAffected()
	return rows, nil
}
