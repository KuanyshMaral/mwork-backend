package moderation

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// ReportReason represents the category of a report
type ReportReason string

const (
	ReportReasonSpam   ReportReason = "spam"
	ReportReasonAbuse  ReportReason = "abuse"
	ReportReasonScam   ReportReason = "scam"
	ReportReasonNudity ReportReason = "nudity"
	ReportReasonOther  ReportReason = "other"
)

// ReportStatus represents the status of a report
type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "pending"
	ReportStatusReviewing ReportStatus = "reviewing"
	ReportStatusResolved  ReportStatus = "resolved"
	ReportStatusDismissed ReportStatus = "dismissed"
)

// UserBlock represents a blocking relationship between users
type UserBlock struct {
	ID            uuid.UUID `db:"id" json:"id"`
	BlockerUserID uuid.UUID `db:"blocker_user_id" json:"blocker_user_id"`
	BlockedUserID uuid.UUID `db:"blocked_user_id" json:"blocked_user_id"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

// UserReport represents a user-generated report for moderation
type UserReport struct {
	ID             uuid.UUID      `db:"id" json:"id"`
	ReporterUserID uuid.UUID      `db:"reporter_user_id" json:"reporter_user_id"`
	ReportedUserID uuid.UUID      `db:"reported_user_id" json:"reported_user_id"`
	RoomID         uuid.NullUUID  `db:"room_id" json:"room_id,omitempty"`
	MessageID      uuid.NullUUID  `db:"message_id" json:"message_id,omitempty"`
	Reason         ReportReason   `db:"reason" json:"reason"`
	Description    sql.NullString `db:"description" json:"description,omitempty"`
	Status         ReportStatus   `db:"status" json:"status"`
	AdminNotes     sql.NullString `db:"admin_notes" json:"admin_notes,omitempty"`
	CreatedAt      time.Time      `db:"created_at" json:"created_at"`
	ResolvedAt     sql.NullTime   `db:"resolved_at" json:"resolved_at,omitempty"`
}
