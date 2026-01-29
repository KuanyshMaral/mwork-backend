package moderation

import "github.com/google/uuid"

// BlockUserRequest represents request to block a user
type BlockUserRequest struct {
	UserID uuid.UUID `json:"user_id" validate:"required"`
}

// CreateReportRequest represents request to report a user
type CreateReportRequest struct {
	ReportedUserID uuid.UUID    `json:"reported_user_id" validate:"required"`
	RoomID         *uuid.UUID   `json:"room_id,omitempty"`
	MessageID      *uuid.UUID   `json:"message_id,omitempty"`
	Reason         ReportReason `json:"reason" validate:"required,oneof=spam abuse scam nudity other"`
	Description    string       `json:"description,omitempty" validate:"max=1000"`
}

// ResolveReportRequest represents admin action on a report
type ResolveReportRequest struct {
	Action string `json:"action" validate:"required,oneof=warn suspend delete dismiss"`
	Notes  string `json:"notes,omitempty" validate:"max=1000"`
}

// BlockedUserResponse represents a blocked user with details
type BlockedUserResponse struct {
	UserID    uuid.UUID `json:"user_id"`
	Username  string    `json:"username"`
	FullName  string    `json:"full_name,omitempty"`
	BlockedAt string    `json:"blocked_at"`
}

// ReportResponse represents a report with full details
type ReportResponse struct {
	*UserReport
	ReporterUsername string `json:"reporter_username,omitempty"`
	ReportedUsername string `json:"reported_username,omitempty"`
}

// ListReportsFilter for filtering reports in admin panel
type ListReportsFilter struct {
	Status ReportStatus `json:"status,omitempty"`
	Limit  int          `json:"limit,omitempty"`
	Offset int          `json:"offset,omitempty"`
}
