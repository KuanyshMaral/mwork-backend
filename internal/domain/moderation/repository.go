package moderation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines moderation data access interface
type Repository interface {
	// Block operations
	CreateBlock(ctx context.Context, block *UserBlock) error
	DeleteBlock(ctx context.Context, blockerID, blockedID uuid.UUID) error
	GetBlock(ctx context.Context, blockerID, blockedID uuid.UUID) (*UserBlock, error)
	ListBlocksByUser(ctx context.Context, userID uuid.UUID) ([]*UserBlock, error)
	IsBlocked(ctx context.Context, user1, user2 uuid.UUID) (bool, error)

	// Report operations
	CreateReport(ctx context.Context, report *UserReport) error
	GetReportByID(ctx context.Context, id uuid.UUID) (*UserReport, error)
	ListReportsByReporter(ctx context.Context, reporterID uuid.UUID) ([]*UserReport, error)
	ListReports(ctx context.Context, filter *ListReportsFilter) ([]*UserReport, error)
	UpdateReportStatus(ctx context.Context, id uuid.UUID, status ReportStatus, adminNotes string) error
	CountReports(ctx context.Context, filter *ListReportsFilter) (int, error)
}

type repository struct {
	db *sqlx.DB
}

// NewRepository creates new moderation repository
func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

// Block operations

func (r *repository) CreateBlock(ctx context.Context, block *UserBlock) error {
	query := `
		INSERT INTO user_blocks (id, blocker_user_id, blocked_user_id, created_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.db.ExecContext(ctx, query,
		block.ID,
		block.BlockerUserID,
		block.BlockedUserID,
		block.CreatedAt,
	)
	return err
}

func (r *repository) DeleteBlock(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	query := `DELETE FROM user_blocks WHERE blocker_user_id = $1 AND blocked_user_id = $2`
	result, err := r.db.ExecContext(ctx, query, blockerID, blockedID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrBlockNotFound
	}

	return nil
}

func (r *repository) GetBlock(ctx context.Context, blockerID, blockedID uuid.UUID) (*UserBlock, error) {
	query := `SELECT * FROM user_blocks WHERE blocker_user_id = $1 AND blocked_user_id = $2`
	var block UserBlock
	err := r.db.GetContext(ctx, &block, query, blockerID, blockedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &block, nil
}

func (r *repository) ListBlocksByUser(ctx context.Context, userID uuid.UUID) ([]*UserBlock, error) {
	query := `
		SELECT * FROM user_blocks 
		WHERE blocker_user_id = $1
		ORDER BY created_at DESC
	`
	var blocks []*UserBlock
	err := r.db.SelectContext(ctx, &blocks, query, userID)
	return blocks, err
}

func (r *repository) IsBlocked(ctx context.Context, user1, user2 uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM user_blocks 
			WHERE (blocker_user_id = $1 AND blocked_user_id = $2)
			   OR (blocker_user_id = $2 AND blocked_user_id = $1)
		)
	`
	var exists bool
	err := r.db.GetContext(ctx, &exists, query, user1, user2)
	return exists, err
}

// Report operations

func (r *repository) CreateReport(ctx context.Context, report *UserReport) error {
	query := `
		INSERT INTO user_reports (
			id, reporter_user_id, reported_user_id, room_id, message_id,
			reason, description, status, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`
	_, err := r.db.ExecContext(ctx, query,
		report.ID,
		report.ReporterUserID,
		report.ReportedUserID,
		report.RoomID,
		report.MessageID,
		report.Reason,
		report.Description,
		report.Status,
		report.CreatedAt,
	)
	return err
}

func (r *repository) GetReportByID(ctx context.Context, id uuid.UUID) (*UserReport, error) {
	query := `SELECT * FROM user_reports WHERE id = $1`
	var report UserReport
	err := r.db.GetContext(ctx, &report, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &report, nil
}

func (r *repository) ListReportsByReporter(ctx context.Context, reporterID uuid.UUID) ([]*UserReport, error) {
	query := `
		SELECT * FROM user_reports 
		WHERE reporter_user_id = $1
		ORDER BY created_at DESC
	`
	var reports []*UserReport
	err := r.db.SelectContext(ctx, &reports, query, reporterID)
	return reports, err
}

func (r *repository) ListReports(ctx context.Context, filter *ListReportsFilter) ([]*UserReport, error) {
	query := `
		SELECT * FROM user_reports
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	if filter != nil {
		if filter.Status != "" {
			query += fmt.Sprintf(` AND status = $%d`, argPos)
			args = append(args, filter.Status)
			argPos++
		}

		query += ` ORDER BY created_at DESC`

		if filter.Limit > 0 {
			query += fmt.Sprintf(` LIMIT $%d`, argPos)
			args = append(args, filter.Limit)
			argPos++
		}

		if filter.Offset > 0 {
			query += fmt.Sprintf(` OFFSET $%d`, argPos)
			args = append(args, filter.Offset)
		}
	} else {
		query += ` ORDER BY created_at DESC LIMIT 50`
	}

	var reports []*UserReport
	err := r.db.SelectContext(ctx, &reports, query, args...)
	return reports, err
}

func (r *repository) UpdateReportStatus(ctx context.Context, id uuid.UUID, status ReportStatus, adminNotes string) error {
	var resolvedAt sql.NullTime
	if status == ReportStatusResolved || status == ReportStatusDismissed {
		resolvedAt = sql.NullTime{Time: time.Now(), Valid: true}
	}

	query := `
		UPDATE user_reports 
		SET status = $1, admin_notes = $2, resolved_at = $3
		WHERE id = $4
	`
	_, err := r.db.ExecContext(ctx, query, status, adminNotes, resolvedAt, id)
	return err
}

func (r *repository) CountReports(ctx context.Context, filter *ListReportsFilter) (int, error) {
	query := `SELECT COUNT(*) FROM user_reports WHERE 1=1`
	args := []interface{}{}

	if filter != nil && filter.Status != "" {
		query += ` AND status = $1`
		args = append(args, filter.Status)
	}

	var count int
	err := r.db.GetContext(ctx, &count, query, args...)
	return count, err
}
